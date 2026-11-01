package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/chjpcojit/agents-management/internal/adapter/runtime/claudecli/protocol"
	"github.com/chjpcojit/agents-management/internal/domain"
)

// Status of one spike case. OBSERVED means the case records behaviour to answer a question.
// Trạng thái của một case spike. OBSERVED nghĩa là case ghi lại hành vi để trả lời một câu hỏi.
type Status string

const (
	StatusPass     Status = "PASS"
	StatusFail     Status = "FAIL"
	StatusObserved Status = "OBSERVED"
	StatusSkipped  Status = "SKIPPED"
)

// CaseResult is what one scenario reports. / CaseResult là những gì một scenario báo cáo.
type CaseResult struct {
	ID       string
	Name     string
	Answers  string // Q-n / R-n this case answers / Q-n, R-n mà case này trả lời
	Status   Status
	Notes    []string
	Samples  map[string]string // label → raw NDJSON excerpt / nhãn → trích NDJSON gốc
	Duration time.Duration
}

func (c *CaseResult) note(format string, a ...any) {
	c.Notes = append(c.Notes, fmt.Sprintf(format, a...))
}

func (c *CaseResult) sample(label string, msgs []timedMessage, kinds ...protocol.Kind) {
	if c.Samples == nil {
		c.Samples = map[string]string{}
	}
	var b strings.Builder
	n := 0
	for _, m := range msgs {
		if len(kinds) > 0 && !containsKind(kinds, m.Msg.Kind) {
			continue
		}
		b.WriteString(truncate(string(m.Msg.Raw), 1500))
		b.WriteByte('\n')
		if n++; n >= 6 {
			break
		}
	}
	if b.Len() > 0 {
		c.Samples[label] = b.String()
	}
}

func (c *CaseResult) fail(format string, a ...any) {
	c.Status = StatusFail
	c.note("FAIL: "+format, a...)
}

func containsKind(ks []protocol.Kind, k protocol.Kind) bool {
	for _, x := range ks {
		if x == k {
			return true
		}
	}
	return false
}

// scenarioEnv is what every scenario gets. / scenarioEnv là những gì mọi scenario nhận được.
type scenarioEnv struct {
	ClaudeBin  string
	Model      string
	SandboxDir string
	SamplesDir string
	T7Wait     time.Duration
	TurnLimit  time.Duration
	Log        *slog.Logger
}

func (e scenarioEnv) harness(policy Policy, timeout time.Duration) (*harness, error) {
	return newHarness(harnessOptions{ClaudeBin: e.ClaudeBin, Model: e.Model, ApprovalTimeout: timeout, Policy: policy, Log: e.Log})
}

// rawFile opens a sample capture file for one case and agent, or returns io.Discard.
// rawFile mở file ghi mẫu cho một case và agent, hoặc trả io.Discard.
func (e scenarioEnv) rawFile(caseID, agentID string) (io.Writer, func()) {
	if e.SamplesDir == "" {
		return nil, func() {}
	}
	if err := os.MkdirAll(e.SamplesDir, 0o750); err != nil {
		return nil, func() {}
	}
	f, err := os.OpenFile(filepath.Join(e.SamplesDir, caseID+"-"+agentID+".ndjson"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600) // #nosec G304 -- path is the operator flag plus fixed case ids / path là flag của operator cộng id case cố định
	if err != nil {
		return nil, func() {}
	}
	return f, func() { _ = f.Close() }
}

// scenario is one spike case. / scenario là một case của spike.
type scenario struct {
	ID      string
	Name    string
	Answers string
	Run     func(ctx context.Context, env scenarioEnv, c *CaseResult) error
}

func scenarios() []scenario {
	return []scenario{
		{"T1", "Auth qua login Max, env scrub", "R-1, R-2, SEC-1, A-1", runT1},
		{"T2", "Multi-turn trên 1 process", "R-5, R-6", runT2},
		{"T3", "Gửi prompt khi turn đang chạy", "Q-1, R-8", runT3},
		{"T4", "Stream realtime, tách tool call / result", "R-7", runT4},
		{"T5", "Approval allow", "Q-2, R-10", runT5},
		{"T6", "Approval deny có lý do", "Q-2, R-10, R-11", runT6},
		{"T7", "Approval timeout → deny", "Q-5, R-10, R-12", runT7},
		{"T8", "Bus tool send_message", "R-13", runT8},
		{"T9", "/context", "Q-3, R-15", runT9},
		{"T10", "/compact", "R-14", runT10},
		{"T11", "/usage", "Q-3, R-15", runT11},
		{"T12", "Interrupt (SIGINT) khi tool chạy lâu", "R-9", runT12},
		{"T13", "Resume session sau khi process tắt", "R-4", runT13},
		{"T14", "Crash an toàn (kill -9)", "R-3", runT14},
		{"T15", "2 agent song song, route đúng agent", "Q-4, SEC-4", runT15},
	}
}

// start is the common prologue: harness + one agent in a fresh sandbox.
// start là phần mở đầu chung: harness + một agent trong sandbox mới.
func start(env scenarioEnv, c *CaseResult, policy Policy, timeout time.Duration, agentID string) (*harness, *agent, string, func(), error) {
	h, err := env.harness(policy, timeout)
	if err != nil {
		return nil, nil, "", nil, err
	}
	dir, err := makeSandbox(env.SandboxDir, strings.ToLower(c.ID))
	if err != nil {
		h.Close()
		return nil, nil, "", nil, err
	}
	raw, closeRaw := env.rawFile(c.ID, agentID)
	a, err := h.StartAgent(agentSpec{ID: agentID, Dir: dir, Raw: raw})
	if err != nil {
		h.Close()
		closeRaw()
		return nil, nil, "", nil, err
	}
	return h, a, dir, func() { h.Close(); closeRaw() }, nil
}

func initOf(msgs []timedMessage) *protocol.Init {
	for _, m := range msgs {
		if m.Msg.Kind == protocol.KindSystemInit && m.Msg.Init != nil {
			return m.Msg.Init
		}
	}
	return nil
}

func authFailed(msgs []timedMessage) bool {
	for _, m := range msgs {
		if m.Msg.Kind == protocol.KindAPIRetry && m.Msg.Retry != nil && m.Msg.Retry.Error == "authentication_failed" {
			return true
		}
	}
	return false
}

func checkResultOK(c *CaseResult, label string, msgs []timedMessage, err error) *protocol.Result {
	if err != nil {
		c.fail("%s: %v", label, err)
		return nil
	}
	r := lastResult(msgs)
	if r == nil {
		c.fail("%s: no result", label)
		return nil
	}
	if r.IsError {
		c.fail("%s: result %s is_error=true: %s", label, r.Subtype, truncate(r.Text, 300))
	}
	return r
}

func runT1(ctx context.Context, env scenarioEnv, c *CaseResult) error {
	run := func(label string) error {
		_, a, _, done, err := start(env, c, AllowAll, 0, "t1"+label)
		if err != nil {
			return err
		}
		defer done()
		msgs, err := a.Turn(ctx, "Reply with exactly: PONG", env.TurnLimit)
		if authFailed(msgs) {
			c.fail("%s: api_retry authentication_failed: log in with `claude /login` using the Max account", label)
			return nil
		}
		r := checkResultOK(c, label, msgs, err)
		in := initOf(msgs)
		if in == nil {
			c.fail("%s: no system/init seen", label)
		} else {
			c.note("%s: init model=%s apiKeySource=%q permissionMode=%s tools=%d", label, in.Model, in.APIKeySource, in.PermissionMode, len(in.Tools))
			if in.APIKeySource != "" && in.APIKeySource != "none" {
				c.fail("%s: apiKeySource=%q, the agent is not on the Max login", label, in.APIKeySource)
			}
			for _, s := range in.MCPServers {
				c.note("%s: mcp server %s status=%s", label, s.Name, s.Status)
				if s.Name == "orchestrator" && s.Status != "connected" {
					c.fail("%s: orchestrator MCP server status %s", label, s.Status)
				}
			}
		}
		if r != nil {
			c.note("%s: result=%q cost=$%.4f", label, truncate(r.Text, 80), r.TotalCostUSD)
		}
		c.sample(label+" init", msgs, protocol.KindSystemInit)
		return nil
	}

	if err := run("plain"); err != nil {
		return err
	}
	// Prove the env scrub: a fake API key in the parent env must not reach the agent.
	// Chứng minh env scrub: API key giả trong env cha không được tới agent.
	prev, had := os.LookupEnv("ANTHROPIC_API_KEY")
	if err := os.Setenv("ANTHROPIC_API_KEY", "sk-ant-api03-spike-fake-0000000000"); err != nil {
		return err
	}
	defer func() {
		if had {
			_ = os.Setenv("ANTHROPIC_API_KEY", prev)
		} else {
			_ = os.Unsetenv("ANTHROPIC_API_KEY")
		}
	}()
	return run("fakekey")
}

func runT2(ctx context.Context, env scenarioEnv, c *CaseResult) error {
	_, a, _, done, err := start(env, c, AllowAll, 0, "t2")
	if err != nil {
		return err
	}
	defer done()
	pid := a.r.PID()
	prompts := []string{
		"Remember this code word for later: ZEBRA-42. Reply with only: OK",
		"What is 2+2? Reply with only the number.",
		"What was the code word I gave you? Reply with only the code word.",
	}
	var last *protocol.Result
	for i, p := range prompts {
		msgs, err := a.Turn(ctx, p, env.TurnLimit)
		last = checkResultOK(c, fmt.Sprintf("turn %d", i+1), msgs, err)
		if last == nil {
			return nil
		}
	}
	if !strings.Contains(strings.ToUpper(last.Text), "ZEBRA-42") {
		c.fail("turn 3 did not recall the code word: %q", truncate(last.Text, 200))
	}
	if a.Exited() != nil || a.r.PID() != pid {
		c.fail("process changed or exited between turns")
	}
	c.note("3 results on pid %d, session %s", pid, a.r.SessionID())
	return nil
}

func runT3(ctx context.Context, env scenarioEnv, c *CaseResult) error {
	_, a, _, done, err := start(env, c, AllowAll, 0, "t3")
	if err != nil {
		return err
	}
	defer done()
	c.Status = StatusObserved
	from := a.Len()
	if err := a.r.Send("Write the numbers from 1 to 40, one per line, with a short fact about each number."); err != nil {
		return err
	}
	wctx, cancel := context.WithTimeout(ctx, env.TurnLimit)
	defer cancel()
	if _, err := a.WaitFor(wctx, from, func(m []timedMessage) bool {
		for _, x := range m {
			if x.Msg.TextDelta != "" {
				return true
			}
		}
		return false
	}); err != nil {
		c.fail("turn 1 never streamed: %v", err)
		return nil
	}
	sentAt := time.Now()
	if err := a.r.ForceSend("Stop. Reply with exactly: SECOND"); err != nil {
		c.fail("force send while running: %v", err)
		return nil
	}
	msgs, err := a.WaitFor(wctx, from, hasResults(2))
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		c.fail("waiting for results: %v", err)
	}
	var results []string
	for _, m := range msgs {
		if m.Msg.Kind == protocol.KindResult && m.Msg.Result != nil {
			results = append(results, fmt.Sprintf("%s after %s: %q", m.Msg.Result.Subtype, m.At.Sub(sentAt).Round(time.Millisecond), truncate(m.Msg.Result.Text, 120)))
		}
	}
	c.note("results after the second prompt was written: %d", len(results))
	for i, r := range results {
		c.note("result %d: %s", i+1, r)
	}
	switch len(results) {
	case 0:
		c.fail("no result at all")
	case 1:
		c.note("one result only: the second prompt was merged into the running turn (or dropped); check whether result 1 says SECOND")
	default:
		c.note("two results: the second prompt was queued after turn 1")
	}
	if a.Exited() != nil {
		c.fail("process exited")
	}
	return nil
}

func runT4(ctx context.Context, env scenarioEnv, c *CaseResult) error {
	_, a, dir, done, err := start(env, c, AllowAll, 0, "t4")
	if err != nil {
		return err
	}
	defer done()
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("spike-first-line\nsecond\n"), 0o600); err != nil {
		return err
	}
	msgs, err := a.Turn(ctx, "Use the Read tool to read hello.txt in the current directory, then tell me its first line.", env.TurnLimit)
	if checkResultOK(c, "turn", msgs, err) == nil {
		return nil
	}
	var firstDelta, resultAt time.Time
	uses := map[string]string{}
	var matched int
	for _, m := range msgs {
		if m.Msg.TextDelta != "" && firstDelta.IsZero() {
			firstDelta = m.At
		}
		if m.Msg.Kind == protocol.KindResult {
			resultAt = m.At
		}
		for _, b := range m.Msg.Blocks {
			if b.Type == "tool_use" {
				uses[b.ID] = b.Name
			}
			if b.Type == "tool_result" {
				if _, ok := uses[b.ToolUseID]; ok {
					matched++
				}
			}
		}
	}
	switch {
	case firstDelta.IsZero():
		c.fail("no text delta stream_event")
	case !firstDelta.Before(resultAt):
		c.fail("text delta did not arrive before result")
	default:
		c.note("first text delta %s before result", resultAt.Sub(firstDelta).Round(time.Millisecond))
	}
	if len(uses) == 0 || matched == 0 {
		c.fail("tool_use / tool_result pair not found (uses=%d matched=%d)", len(uses), matched)
	} else {
		c.note("tool_use ids=%d, tool_result matched by tool_use_id=%d", len(uses), matched)
	}
	c.sample("tool_use", msgs, protocol.KindAssistant)
	c.sample("tool_result", msgs, protocol.KindUser)
	return nil
}

// bashPrompt asks for exactly one Bash call. / bashPrompt yêu cầu đúng một lần gọi Bash.
func bashPrompt(command string) string {
	return "Run exactly this shell command with the Bash tool, once, and nothing else: " + command +
		"\nIf it is denied, do not retry and do not use another tool; just tell me the reason you were given."
}

func describeApprovals(c *CaseResult, h *harness, agentID string) []domain.Approval {
	opened := h.Opened(domain.AgentID(agentID))
	for i, a := range opened {
		c.note("approval %d: tool=%s risk=%s", i+1, a.ToolName, a.Risk)
	}
	for i, keys := range h.ArgKeys(domain.AgentID(agentID)) {
		c.note("approval_prompt call %d argument keys: %v", i+1, keys)
	}
	return opened
}

func runT5(ctx context.Context, env scenarioEnv, c *CaseResult) error {
	h, a, dir, done, err := start(env, c, AllowAll, 0, "t5")
	if err != nil {
		return err
	}
	defer done()
	msgs, err := a.Turn(ctx, bashPrompt("touch spike_allow.txt"), env.TurnLimit)
	checkResultOK(c, "turn", msgs, err)
	opened := describeApprovals(c, h, "t5")
	if len(opened) == 0 {
		c.fail("approval_prompt was never called")
	} else if opened[0].ToolName != "Bash" {
		c.fail("first approval was for %s, expected Bash", opened[0].ToolName)
	}
	if _, err := os.Stat(filepath.Join(dir, "spike_allow.txt")); err != nil {
		c.fail("file was not created after allow (the allow response shape may be wrong, Q-2)")
	} else {
		c.note("file created: the {behavior: allow, updatedInput} shape works")
	}
	c.sample("tool_result", msgs, protocol.KindUser)
	return nil
}

func runT6(ctx context.Context, env scenarioEnv, c *CaseResult) error {
	const reason = "spike T6: creating files is not allowed here"
	h, a, dir, done, err := start(env, c, func(domain.Approval) (domain.Decision, string, bool) {
		return domain.DecisionDeny, reason, true
	}, 0, "t6")
	if err != nil {
		return err
	}
	defer done()
	msgs, err := a.Turn(ctx, bashPrompt("touch spike_deny.txt"), env.TurnLimit)
	r := checkResultOK(c, "turn", msgs, err)
	opened := describeApprovals(c, h, "t6")
	if len(opened) == 0 {
		c.fail("approval_prompt was never called")
	}
	if len(opened) > 2 {
		c.fail("%d approvals: Claude retried after deny", len(opened))
	}
	if _, err := os.Stat(filepath.Join(dir, "spike_deny.txt")); err == nil {
		c.fail("file exists although the call was denied")
	}
	if r != nil {
		c.note("final answer: %q", truncate(r.Text, 300))
		if !strings.Contains(strings.ToLower(r.Text), "not allowed") {
			c.note("the final answer does not quote the deny reason; check the tool_result sample")
		}
	}
	c.sample("tool_result", msgs, protocol.KindUser)
	return nil
}

func runT7(ctx context.Context, env scenarioEnv, c *CaseResult) error {
	never := func(domain.Approval) (domain.Decision, string, bool) { return "", "", false }
	h, a, dir, done, err := start(env, c, never, env.T7Wait, "t7")
	if err != nil {
		return err
	}
	defer done()
	sent := time.Now()
	msgs, err := a.Turn(ctx, bashPrompt("touch spike_timeout.txt"), env.T7Wait+env.TurnLimit)
	elapsed := time.Since(sent)
	checkResultOK(c, "turn", msgs, err)
	opened := describeApprovals(c, h, "t7")
	if len(opened) == 0 {
		c.fail("approval_prompt was never called")
		return nil
	}
	fctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	final, err := h.Final(fctx, opened[0].ID)
	if err != nil {
		c.fail("read final approval: %v", err)
	} else if final.Status != domain.ApprovalExpired {
		c.fail("approval status %s, expected expired", final.Status)
	}
	if _, err := os.Stat(filepath.Join(dir, "spike_timeout.txt")); err == nil {
		c.fail("file exists although nobody approved")
	}
	c.note("approval deadline %s; turn ended %s after the prompt", env.T7Wait, elapsed.Round(time.Second))
	for _, m := range msgs {
		for _, b := range m.Msg.Blocks {
			if b.Type == "tool_result" && b.IsError && m.At.Sub(opened[0].CreatedAt) < env.T7Wait-2*time.Second {
				c.note("Q-5: Claude Code ended the tool call after %s, before our deadline: that is the ceiling for ORCH_APPROVAL_TIMEOUT",
					m.At.Sub(opened[0].CreatedAt).Round(time.Second))
			}
		}
	}
	c.sample("tool_result", msgs, protocol.KindUser)
	return nil
}

func runT8(ctx context.Context, env scenarioEnv, c *CaseResult) error {
	h, a, _, done, err := start(env, c, AllowAll, 0, "t8")
	if err != nil {
		return err
	}
	defer done()
	msgs, err := a.Turn(ctx, `Call the tool mcp__orchestrator__send_message once with to="reviewer" and text="hello from spike". Then reply with only: DONE`, env.TurnLimit)
	checkResultOK(c, "turn", msgs, err)
	describeApprovals(c, h, "t8")
	got := h.Messages()
	if len(got) == 0 {
		c.fail("send_message never reached the bus")
		return nil
	}
	m := got[0]
	c.note("bus message from=%s to=%s text=%q", m.From, m.To, truncate(m.Text, 80))
	if m.From != "t8" || m.To != "reviewer" || !strings.Contains(m.Text, "hello") {
		c.fail("unexpected payload")
	}
	return nil
}

func slashCase(cmd string, warmup bool) func(context.Context, scenarioEnv, *CaseResult) error {
	return func(ctx context.Context, env scenarioEnv, c *CaseResult) error {
		_, a, _, done, err := start(env, c, AllowAll, 0, strings.ToLower(c.ID))
		if err != nil {
			return err
		}
		defer done()
		if warmup {
			for _, p := range []string{"Explain in about 150 words what a mutex is.", "Now explain a semaphore in about 150 words."} {
				msgs, err := a.Turn(ctx, p, env.TurnLimit)
				if checkResultOK(c, "warmup", msgs, err) == nil {
					return nil
				}
			}
		} else {
			// Slash commands may need an initialised session. / Slash command có thể cần session đã khởi tạo.
			msgs, err := a.Turn(ctx, "Reply with exactly: READY", env.TurnLimit)
			if checkResultOK(c, "warmup", msgs, err) == nil {
				return nil
			}
		}
		from := a.Len()
		msgs, err := a.Turn(ctx, cmd, env.TurnLimit)
		if err != nil {
			c.fail("%s: %v", cmd, err)
			return nil
		}
		kinds := map[protocol.Kind]int{}
		for _, m := range msgs {
			kinds[m.Msg.Kind]++
		}
		c.note("%s produced %d lines: %v", cmd, len(msgs), kinds)
		if r := lastResult(msgs); r != nil {
			c.note("result subtype=%s is_error=%t text=%q", r.Subtype, r.IsError, truncate(r.Text, 600))
		}
		c.sample(cmd+" output", a.Since(from))
		return nil
	}
}

var (
	runT9  = observe(slashCase("/context", false), "context")
	runT11 = observe(slashCase("/usage", false), "usage")
)

func observe(f func(context.Context, scenarioEnv, *CaseResult) error, keyword string) func(context.Context, scenarioEnv, *CaseResult) error {
	return func(ctx context.Context, env scenarioEnv, c *CaseResult) error {
		c.Status = StatusObserved
		if err := f(ctx, env, c); err != nil {
			return err
		}
		if c.Status == StatusObserved {
			c.note("read the sample: does it hold numbers usable for the %s panel? (answer Q-3 from it)", keyword)
		}
		return nil
	}
}

func runT10(ctx context.Context, env scenarioEnv, c *CaseResult) error {
	if err := slashCase("/compact", true)(ctx, env, c); err != nil {
		return err
	}
	found := false
	for _, s := range c.Samples {
		if strings.Contains(s, `"compact_boundary"`) {
			found = true
			if !strings.Contains(s, "pre_tokens") {
				c.fail("compact_boundary has no pre_tokens")
			}
		}
	}
	if !found {
		c.fail("no compact_boundary message after /compact")
	}
	return nil
}

func runT12(ctx context.Context, env scenarioEnv, c *CaseResult) error {
	if runtime.GOOS == "windows" {
		c.Status = StatusSkipped
		c.note("SIGINT is not available on Windows")
		return nil
	}
	h, a, _, done, err := start(env, c, AllowAll, 0, "t12")
	if err != nil {
		return err
	}
	defer done()
	from := a.Len()
	if err := a.r.Send(bashPrompt("sleep 60")); err != nil {
		return err
	}
	wctx, cancel := context.WithTimeout(ctx, env.TurnLimit)
	defer cancel()
	// Wait until the Bash call is approved, then give it time to start.
	// Chờ tới khi lệnh Bash được duyệt, rồi đợi thêm cho nó chạy.
	for len(h.Opened("t12")) == 0 {
		select {
		case <-wctx.Done():
			c.fail("Bash was never requested")
			return nil
		case <-time.After(200 * time.Millisecond):
		}
	}
	time.Sleep(5 * time.Second)
	intAt := time.Now()
	if err := a.r.Interrupt(); err != nil {
		c.fail("interrupt: %v", err)
		return nil
	}
	ictx, icancel := context.WithTimeout(ctx, 30*time.Second)
	defer icancel()
	msgs, err := a.WaitFor(ictx, from, hasResults(1))
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		c.fail("no result within 30s of SIGINT")
	case err != nil:
		c.fail("after SIGINT: %v (exit %v)", err, exitCode(a))
		return nil
	default:
		r := lastResult(msgs)
		c.note("result %s is_error=%t %s after SIGINT", r.Subtype, r.IsError, time.Since(intAt).Round(time.Millisecond))
	}
	c.sample("after interrupt", msgs, protocol.KindResult, protocol.KindUser)
	next, err := a.Turn(ctx, "Reply with exactly: AFTER", env.TurnLimit)
	if r := checkResultOK(c, "turn after interrupt", next, err); r != nil {
		c.note("process alive and answered: %q", truncate(r.Text, 40))
	}
	cctx, ccancel := context.WithTimeout(ctx, 10*time.Second)
	defer ccancel()
	_ = a.r.Close(cctx)
	c.note("graceful close (stdin EOF) exit code %d; SIGTERM would give 143", a.r.ExitCode())
	return nil
}

func exitCode(a *agent) any {
	if e := a.Exited(); e != nil {
		return e.ExitCode
	}
	return "running"
}

func runT13(ctx context.Context, env scenarioEnv, c *CaseResult) error {
	h, err := env.harness(AllowAll, 0)
	if err != nil {
		return err
	}
	defer h.Close()
	dir, err := makeSandbox(env.SandboxDir, "t13")
	if err != nil {
		return err
	}
	first, err := h.StartAgent(agentSpec{ID: "t13", Dir: dir})
	if err != nil {
		return err
	}
	msgs, err := first.Turn(ctx, "Remember this code word for later: OTTER-7. Reply with only: OK", env.TurnLimit)
	if checkResultOK(c, "turn 1", msgs, err) == nil {
		return nil
	}
	sid := first.r.SessionID()
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := first.r.Close(cctx); err != nil {
		return err
	}
	c.note("first process exited with %d, session %s", first.r.ExitCode(), sid)

	second, err := h.StartAgent(agentSpec{ID: "t13", Dir: dir, Resume: sid})
	if err != nil {
		return err
	}
	msgs, err = second.Turn(ctx, "What was the code word I gave you? Reply with only the code word.", env.TurnLimit)
	r := checkResultOK(c, "resumed turn", msgs, err)
	if r != nil && !strings.Contains(strings.ToUpper(r.Text), "OTTER-7") {
		c.fail("resumed session did not recall the code word: %q", truncate(r.Text, 200))
	}
	if got := second.r.SessionID(); got != sid {
		c.note("session id changed on resume: %s → %s", sid, got)
	}
	return nil
}

func runT14(ctx context.Context, env scenarioEnv, c *CaseResult) error {
	before := runtime.NumGoroutine()
	h, a, _, done, err := start(env, c, AllowAll, 0, "t14")
	if err != nil {
		return err
	}
	if err := a.r.Send(bashPrompt("sleep 30")); err != nil {
		done()
		return err
	}
	wctx, cancel := context.WithTimeout(ctx, env.TurnLimit)
	defer cancel()
	for len(h.Opened("t14")) == 0 {
		select {
		case <-wctx.Done():
			done()
			c.fail("Bash was never requested")
			return nil
		case <-time.After(200 * time.Millisecond):
		}
	}
	time.Sleep(2 * time.Second)
	p, err := os.FindProcess(a.r.PID())
	if err != nil {
		done()
		return err
	}
	if err := p.Kill(); err != nil {
		done()
		return fmt.Errorf("kill -9: %w", err)
	}
	select {
	case <-a.drained:
		e := a.Exited()
		c.note("exit event: state=%s code=%d", e.State, e.ExitCode)
	case <-time.After(10 * time.Second):
		c.fail("no exit event within 10s of kill")
	}
	pctx, pcancel := context.WithTimeout(ctx, 5*time.Second)
	defer pcancel()
	if code, err := h.Probe(pctx); err != nil || code != 401 {
		c.fail("MCP server not answering after crash: code=%d err=%v", code, err)
	} else {
		c.note("MCP server still answers (401 without token)")
	}
	done()
	time.Sleep(500 * time.Millisecond)
	after := runtime.NumGoroutine()
	c.note("goroutines before=%d after=%d", before, after)
	if after > before+5 {
		c.fail("possible goroutine leak: %d → %d", before, after)
	}
	return nil
}

func runT15(ctx context.Context, env scenarioEnv, c *CaseResult) error {
	h, err := env.harness(AllowAll, 0)
	if err != nil {
		return err
	}
	defer h.Close()
	type pair struct {
		id  string
		dir string
		a   *agent
	}
	agents := []*pair{{id: "alpha"}, {id: "beta"}}
	for _, p := range agents {
		if p.dir, err = makeSandbox(env.SandboxDir, "t15-"+p.id); err != nil {
			return err
		}
		raw, closeRaw := env.rawFile(c.ID, p.id)
		defer closeRaw()
		if p.a, err = h.StartAgent(agentSpec{ID: p.id, Dir: p.dir, Raw: raw}); err != nil {
			return err
		}
	}
	errs := make(chan error, len(agents))
	for _, p := range agents {
		go func(p *pair) {
			prompt := bashPrompt("touch from_"+p.id+".txt") +
				"\nAfter that, call the tool mcp__orchestrator__send_message once with to=\"board\" and text=\"" + p.id + " done\"."
			msgs, err := p.a.Turn(ctx, prompt, env.TurnLimit)
			if err == nil && lastResult(msgs) == nil {
				err = errors.New("no result")
			}
			if err != nil {
				err = fmt.Errorf("%s: %w", p.id, err)
			}
			errs <- err
		}(p)
	}
	for range agents {
		if err := <-errs; err != nil {
			c.fail("%v", err)
		}
	}
	for _, p := range agents {
		other := "alpha"
		if p.id == "alpha" {
			other = "beta"
		}
		for _, ap := range h.Opened(domain.AgentID(p.id)) {
			if strings.Contains(string(ap.Input), "from_"+other) {
				c.fail("approval for %s carried %s's command", p.id, other)
			}
		}
		if n := len(h.Opened(domain.AgentID(p.id))); n == 0 {
			c.fail("no approval routed to %s", p.id)
		} else {
			c.note("%s: %d approvals routed by URL path /mcp/%s", p.id, n, p.id)
		}
		if _, err := os.Stat(filepath.Join(p.dir, "from_"+p.id+".txt")); err != nil {
			c.fail("%s: file not created in its own sandbox", p.id)
		}
		if _, err := os.Stat(filepath.Join(p.dir, "from_"+other+".txt")); err == nil {
			c.fail("%s's sandbox holds %s's file", p.id, other)
		}
	}
	for _, m := range h.Messages() {
		c.note("bus: from=%s to=%s text=%q", m.From, m.To, truncate(m.Text, 60))
		if !strings.HasPrefix(m.Text, string(m.From)) {
			c.fail("bus message from %s carries another agent's text", m.From)
		}
	}
	return nil
}
