package claudecli_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/chjpcojit/agents-management/internal/adapter/runtime/claudecli"
	"github.com/chjpcojit/agents-management/internal/adapter/runtime/claudecli/protocol"
)

const sid = "11111111-1111-4111-8111-111111111111"

func start(t *testing.T, mutate func(*claudecli.Config)) *claudecli.Runner {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cfg := claudecli.Config{
		AgentID: "agent-a", Bin: self, Dir: t.TempDir(), SessionID: sid,
		MCPURL: "http://127.0.0.1:1/mcp/agent-a", MCPToken: "tok",
		ExtraEnv:          []string{"FAKE_CLAUDE=1"},
		FirstEventTimeout: 5 * time.Second,
	}
	if mutate != nil {
		mutate(&cfg)
	}
	r, err := claudecli.Start(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		go drain(r)
		_ = r.Close(ctx)
	})
	return r
}

func drain(r *claudecli.Runner) {
	for range r.Events() {
	}
}

// nextResult reads events until a result message and returns everything read.
// nextResult đọc event tới khi gặp result và trả về toàn bộ event đã đọc.
func nextResult(t *testing.T, r *claudecli.Runner) []claudecli.Event {
	t.Helper()
	var got []claudecli.Event
	timeout := time.After(10 * time.Second)
	for {
		select {
		case e, ok := <-r.Events():
			if !ok {
				t.Fatalf("events closed before result; got %d events", len(got))
			}
			got = append(got, e)
			if e.Type == claudecli.EventMessage && e.Message.Kind == protocol.KindResult {
				return got
			}
		case <-timeout:
			t.Fatal("timed out waiting for result")
		}
	}
}

func resultText(evs []claudecli.Event) string {
	return evs[len(evs)-1].Message.Result.Text
}

func waitState(t *testing.T, r *claudecli.Runner, want claudecli.State) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for r.State() != want {
		if time.Now().After(deadline) {
			t.Fatalf("want state %s, got %s", want, r.State())
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestStartValidatesConfig(t *testing.T) {
	_, err := claudecli.Start(claudecli.Config{Dir: "/nope"}, nil)
	if !errors.Is(err, claudecli.ErrInvalidConfig) {
		t.Fatalf("want ErrInvalidConfig, got %v", err)
	}
	_, err = claudecli.Start(claudecli.Config{AgentID: "a", Dir: t.TempDir(), SessionID: "s", Resume: "r", MCPURL: "u", MCPToken: "t"}, nil)
	if !errors.Is(err, claudecli.ErrInvalidConfig) {
		t.Fatalf("session id and resume together must be rejected, got %v", err)
	}
}

func TestBuildArgs(t *testing.T) {
	args, err := claudecli.BuildArgs(claudecli.Config{SessionID: sid, MCPURL: "http://127.0.0.1:9/mcp/a", MCPToken: "tok", Model: "sonnet"})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	for _, want := range []string{
		"-p", "--input-format stream-json", "--output-format stream-json", "--verbose",
		"--strict-mcp-config", "--permission-prompt-tool " + claudecli.PermissionPromptTool,
		"--session-id " + sid, "--model sonnet", `"crossSessionInbound":"refuse"`, `"Authorization":"Bearer tok"`,
		`"ask":["Bash","Edit","Write"`,
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("args missing %q", want)
		}
	}
	if slices.Contains(args, "--bare") {
		t.Error("--bare must never be set: it ignores the subscription login")
	}
	args, _ = claudecli.BuildArgs(claudecli.Config{Resume: sid, MCPURL: "u", MCPToken: "t"})
	if !strings.Contains(strings.Join(args, " "), "--resume "+sid) {
		t.Error("resume flag missing")
	}
}

func TestEnvAllowlistDropsAPIKeys(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-fake")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "fake")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "fake")
	r := start(t, nil)
	if err := r.Send("env"); err != nil {
		t.Fatal(err)
	}
	env := resultText(nextResult(t, r))
	for _, bad := range []string{"ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN", "AWS_SECRET_ACCESS_KEY"} {
		if strings.Contains(env, bad) {
			t.Errorf("child env leaked %s", bad)
		}
	}
	if !strings.Contains(env, "HOME=") || !strings.Contains(env, "PATH=") {
		t.Error("child env must keep HOME and PATH")
	}
}

func TestMultiTurnAndStateMachine(t *testing.T) {
	r := start(t, nil)
	pid := r.PID()
	if r.State() != claudecli.StateIdle {
		t.Fatalf("want idle after start, got %s", r.State())
	}
	for i, p := range []string{"one", "two", "three"} {
		if err := r.Send(p); err != nil {
			t.Fatalf("turn %d: %v", i, err)
		}
		evs := nextResult(t, r)
		if got := resultText(evs); got != "echo: "+p {
			t.Fatalf("turn %d: got %q", i, got)
		}
		if i == 0 {
			if evs[0].Type != claudecli.EventState || evs[0].State != claudecli.StateRunning {
				t.Fatalf("first event must be running state, got %+v", evs[0])
			}
			kinds := map[protocol.Kind]bool{}
			for _, e := range evs {
				kinds[e.Message.Kind] = true
			}
			if !kinds[protocol.KindSystemInit] || !kinds[protocol.KindStreamEvent] || !kinds[protocol.KindAssistant] {
				t.Fatalf("missing kinds: %v", kinds)
			}
		}
		waitState(t, r, claudecli.StateIdle)
	}
	if r.PID() != pid || r.SessionID() != sid {
		t.Fatal("must stay on one process and session")
	}
}

// Claude Code queues a prompt written during a turn (spike T3); the runner stays running until both results.
// Claude Code xếp hàng prompt ghi vào giữa turn (spike T3); runner ở running tới khi có đủ hai result.
func TestSendWhileRunningQueues(t *testing.T) {
	r := start(t, nil)
	if err := r.Send("sleep 1"); err != nil {
		t.Fatal(err)
	}
	if err := r.Send("again"); err != nil {
		t.Fatalf("second send while running: %v", err)
	}
	if got := r.Pending(); got != 2 {
		t.Fatalf("pending = %d, want 2", got)
	}
	// Check the event order, not State(): the reader may already be past the second result.
	// Kiểm thứ tự event, không đọc State(): reader có thể đã đọc qua result thứ hai.
	first := nextResult(t, r)
	if got := resultText(first); got != "slept" {
		t.Fatalf("first result %q", got)
	}
	second := nextResult(t, r)
	if got := resultText(second); got != "echo: again" {
		t.Fatalf("second result %q", got)
	}
	for _, e := range append(first, second...) {
		if e.Type == claudecli.EventState && e.State == claudecli.StateIdle {
			t.Fatal("runner went idle before the second result")
		}
	}
	waitState(t, r, claudecli.StateIdle)
	if got := r.Pending(); got != 0 {
		t.Fatalf("pending = %d, want 0", got)
	}
}

func TestGarbageLineKeepsStream(t *testing.T) {
	r := start(t, nil)
	_ = r.Send("garbage")
	evs := nextResult(t, r)
	var unknown int
	for _, e := range evs {
		if e.Message.Kind == protocol.KindUnknown && e.Message.DecodeError != nil {
			unknown++
		}
	}
	if unknown != 1 || resultText(evs) != "after garbage" {
		t.Fatalf("want 1 unknown then result, got %d / %q", unknown, resultText(evs))
	}
}

func TestInterruptEndsTurnAndKeepsProcess(t *testing.T) {
	r := start(t, nil)
	_ = r.Send("sleep 30")
	// Wait for the tool_use so the fake is inside its sleep. / Chờ tool_use để fake đang ở trong sleep.
	for e := range r.Events() {
		if e.Message.Kind == protocol.KindAssistant {
			break
		}
	}
	begin := time.Now()
	if err := r.Interrupt(); err != nil {
		t.Fatal(err)
	}
	evs := nextResult(t, r)
	if resultText(evs) != "interrupted" || time.Since(begin) > 5*time.Second {
		t.Fatalf("interrupt did not end the turn promptly: %q", resultText(evs))
	}
	waitState(t, r, claudecli.StateIdle)
	_ = r.Send("still alive")
	if got := resultText(nextResult(t, r)); got != "echo: still alive" {
		t.Fatalf("process must accept prompts after interrupt, got %q", got)
	}
}

func TestCrashEmitsExited(t *testing.T) {
	r := start(t, nil)
	_ = r.Send("crash")
	var last claudecli.Event
	for e := range r.Events() {
		last = e
	}
	if last.Type != claudecli.EventExited || last.ExitCode != 3 || last.State != claudecli.StateStopped {
		t.Fatalf("want exited code 3, got %+v", last)
	}
	if err := r.Send("x"); !errors.Is(err, claudecli.ErrProcessExited) {
		t.Fatalf("want ErrProcessExited, got %v", err)
	}
}

func TestKill9EmitsExited(t *testing.T) {
	r := start(t, nil)
	proc, err := os.FindProcess(r.PID())
	if err != nil {
		t.Fatal(err)
	}
	if err := proc.Kill(); err != nil {
		t.Fatal(err)
	}
	go drain(r)
	select {
	case <-r.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("runner did not notice the killed process")
	}
	if r.State() != claudecli.StateStopped {
		t.Fatalf("want stopped, got %s", r.State())
	}
}

func TestAuthFailureMarksFailed(t *testing.T) {
	r := start(t, nil)
	_ = r.Send("authfail")
	go drain(r)
	select {
	case <-r.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("process not terminated after auth failure")
	}
	if r.State() != claudecli.StateFailed {
		t.Fatalf("want failed, got %s", r.State())
	}
}

func TestSilentPromptFails(t *testing.T) {
	r := start(t, func(c *claudecli.Config) { c.FirstEventTimeout = 300 * time.Millisecond })
	_ = r.Send("hello") // first turn produces output / turn đầu có output
	nextResult(t, r)
	_ = r.Send("silent")
	go drain(r)
	select {
	case <-r.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("silent prompt did not fail the agent")
	}
	if r.State() != claudecli.StateFailed {
		t.Fatalf("want failed, got %s", r.State())
	}
}

func TestCloseExitsCleanly(t *testing.T) {
	r := start(t, nil)
	go drain(r)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := r.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if r.ExitCode() != 0 {
		t.Fatalf("want exit 0 after stdin close, got %d", r.ExitCode())
	}
}
