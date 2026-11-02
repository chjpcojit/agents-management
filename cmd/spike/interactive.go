package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/chjpcojit/agents-management/internal/adapter/runtime/claudecli"
	"github.com/chjpcojit/agents-management/internal/adapter/runtime/claudecli/protocol"
	"github.com/chjpcojit/agents-management/internal/domain"
)

const interactiveHelp = `commands:
  <text>                 send a prompt (queued by Claude Code while a turn runs)
  :stop                  cancel: stop the agent and its commands (resume with -resume)
  :pending               list pending approvals
  :allow <id>            allow an approval (id prefix is enough)
  :deny <id> [reason]    deny an approval
  :sid                   print the session id (use with -resume)
  :quit                  close the agent and exit
`

func cmdInteractive(args []string, stdin io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	bin := fs.String("claude", "claude", "claude binary / binary claude")
	model := fs.String("model", "", "model alias or id, empty = default / model, rỗng = mặc định")
	dir := fs.String("dir", "", "working directory, empty = new temp sandbox / thư mục làm việc, rỗng = sandbox tạm")
	resume := fs.String("resume", "", "session id to resume / session id để resume")
	timeout := fs.Duration("approval-timeout", 5*time.Minute, "deadline per approval, then deny / deadline mỗi approval, quá thì deny")
	rawPath := fs.String("raw", "", "write raw NDJSON to this file / ghi NDJSON gốc vào file này")
	verbose := fs.Bool("v", false, "debug logs on stderr / log debug ra stderr")
	if err := fs.Parse(args); err != nil {
		return err
	}

	workDir := *dir
	var err error
	if workDir == "" {
		workDir, err = makeSandbox("", "run")
	} else {
		workDir, err = claudecli.SandboxDir(workDir)
	}
	if err != nil {
		return err
	}

	var raw io.Writer
	if *rawPath != "" {
		if err := os.MkdirAll(filepath.Dir(*rawPath), 0o750); err != nil {
			return fmt.Errorf("create raw capture dir: %w", err)
		}
		f, err := os.OpenFile(*rawPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			return fmt.Errorf("open raw capture: %w", err)
		}
		defer func() { _ = f.Close() }()
		raw = f
	}

	var outMu sync.Mutex
	printf := func(format string, a ...any) {
		outMu.Lock()
		fmt.Fprintf(stdout, format, a...)
		outMu.Unlock()
	}

	h, err := newHarness(harnessOptions{
		ClaudeBin:       *bin,
		Model:           *model,
		ApprovalTimeout: *timeout,
		Policy:          func(domain.Approval) (domain.Decision, string, bool) { return "", "", false },
		OnApproval: func(a domain.Approval) {
			printf("\n⚠ approval %s  tool=%s risk=%s  deadline=%s\n  input: %s\n  → :allow %s   |   :deny %s <reason>\n",
				short(a.ID), a.ToolName, a.Risk, a.Deadline.Local().Format("15:04:05"), truncate(string(a.Input), 300), short(a.ID), short(a.ID))
		},
		Log: newLogger(*verbose),
	})
	if err != nil {
		return err
	}
	defer h.Close()

	ag, err := h.StartAgent(agentSpec{ID: "spike", Dir: workDir, Resume: *resume, Raw: raw})
	if err != nil {
		return err
	}
	pr := &printer{printf: printf}
	ag.mu.Lock()
	ag.onEvent = pr.event
	ag.mu.Unlock()

	printf("agent started: pid=%d dir=%s session=%s\n%s", ag.r.PID(), workDir, ag.r.SessionID(), interactiveHelp)

	lines := make(chan string)
	go func() {
		defer close(lines)
		sc := bufio.NewScanner(stdin)
		sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
		for sc.Scan() {
			lines <- sc.Text()
		}
	}()

	for {
		select {
		case <-ag.drained:
			if e := ag.Exited(); e != nil {
				printf("agent exited: state=%s code=%d\n", e.State, e.ExitCode)
			}
			return nil
		case line, ok := <-lines:
			if !ok {
				return nil
			}
			if quit := handleLine(context.Background(), h, ag, strings.TrimSpace(line), printf); quit {
				printf("closing agent, session=%s (resume with -resume)\n", ag.r.SessionID())
				return nil
			}
		}
	}
}

func handleLine(ctx context.Context, h *harness, ag *agent, line string, printf func(string, ...any)) (quit bool) {
	if line == "" {
		return false
	}
	cmd, rest, _ := strings.Cut(line, " ")
	switch cmd {
	case ":quit", ":q":
		return true
	case ":help":
		printf("%s", interactiveHelp)
	case ":sid":
		printf("session=%s pid=%d state=%s\n", ag.r.SessionID(), ag.r.PID(), ag.r.State())
	case ":stop":
		sctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := ag.r.Stop(sctx); err != nil {
			printf("stop failed: %v\n", err)
		}
		printf("stopped; continue with: go run ./cmd/spike run -resume %s -dir <same dir>\n", ag.r.SessionID())
	case ":pending":
		list, err := h.approvals.Pending(ctx)
		if err != nil {
			printf("pending: %v\n", err)
			return false
		}
		if len(list) == 0 {
			printf("no pending approvals\n")
		}
		for _, a := range list {
			printf("  %s tool=%s risk=%s left=%s\n", short(a.ID), a.ToolName, a.Risk, time.Until(a.Deadline).Round(time.Second))
		}
	case ":allow", ":deny":
		prefix, reason, _ := strings.Cut(strings.TrimSpace(rest), " ")
		id, err := findPending(ctx, h, prefix)
		if err != nil {
			printf("%v\n", err)
			return false
		}
		d := domain.DecisionAllow
		if cmd == ":deny" {
			d = domain.DecisionDeny
		}
		a, err := h.approvals.Decide(ctx, id, d, "owner", strings.TrimSpace(reason))
		if err != nil {
			printf("decide %s: %v\n", short(id), err)
			return false
		}
		printf("approval %s → %s\n", short(a.ID), a.Status)
	default:
		if strings.HasPrefix(cmd, ":") {
			printf("unknown command %s\n%s", cmd, interactiveHelp)
			return false
		}
		if err := ag.r.Send(line); err != nil {
			printf("send: %v\n", err)
		} else if p := ag.r.Pending(); p > 1 {
			printf("queued: %d prompts waiting for a result\n", p)
		}
	}
	return false
}

func findPending(ctx context.Context, h *harness, prefix string) (domain.ApprovalID, error) {
	if prefix == "" {
		return "", errors.New("usage: :allow <id> | :deny <id> [reason]")
	}
	list, err := h.approvals.Pending(ctx)
	if err != nil {
		return "", err
	}
	var match []domain.ApprovalID
	for _, a := range list {
		if strings.HasPrefix(string(a.ID), prefix) {
			match = append(match, a.ID)
		}
	}
	switch len(match) {
	case 0:
		return "", fmt.Errorf("no pending approval starts with %q", prefix)
	case 1:
		return match[0], nil
	default:
		return "", fmt.Errorf("%q matches %d approvals, type more of the id", prefix, len(match))
	}
}

// printer renders runner events compactly for a human.
// printer hiển thị event của runner gọn gàng cho người đọc.
type printer struct {
	printf    func(string, ...any)
	streaming bool
}

func (p *printer) event(e claudecli.Event) {
	switch e.Type {
	case claudecli.EventState:
		if e.Reason != "" {
			p.endStream()
			p.printf("● state=%s reason=%s\n", e.State, e.Reason)
		}
		return
	case claudecli.EventExited:
		p.endStream()
		return
	}
	m := e.Message
	switch m.Kind {
	case protocol.KindStreamEvent:
		if m.TextDelta != "" {
			p.streaming = true
			p.printf("%s", m.TextDelta)
		}
	case protocol.KindSystemInit:
		p.endStream()
		if m.Init != nil {
			var mcp []string
			for _, s := range m.Init.MCPServers {
				mcp = append(mcp, s.Name+"="+s.Status)
			}
			p.printf("● init model=%s tools=%d mode=%s apiKeySource=%s mcp=[%s]\n",
				m.Init.Model, len(m.Init.Tools), m.Init.PermissionMode, m.Init.APIKeySource, strings.Join(mcp, " "))
		}
	case protocol.KindCompactBoundary:
		p.endStream()
		p.printf("● compact_boundary %s\n", truncate(string(m.Raw), 300))
	case protocol.KindAPIRetry:
		p.endStream()
		if m.Retry != nil {
			p.printf("● api_retry attempt=%d/%d error=%s\n", m.Retry.Attempt, m.Retry.MaxRetries, m.Retry.Error)
		}
	case protocol.KindAssistant, protocol.KindUser:
		for _, b := range m.Blocks {
			switch b.Type {
			case "tool_use":
				p.endStream()
				p.printf("→ %s %s %s\n", b.Name, b.ID, truncate(string(b.Input), 200))
			case "tool_result":
				p.endStream()
				p.printf("← %s error=%t\n", b.ToolUseID, b.IsError)
			case "text":
				if m.Kind == protocol.KindAssistant && !p.streaming {
					p.printf("%s\n", b.Text)
				}
			}
		}
		if m.Kind == protocol.KindAssistant {
			p.endStream()
		}
	case protocol.KindResult:
		p.endStream()
		if r := m.Result; r != nil {
			p.printf("■ result %s error=%t turns=%d %dms cost=$%.4f\n", r.Subtype, r.IsError, r.NumTurns, r.DurationMS, r.TotalCostUSD)
			if r.IsError && r.Text != "" {
				p.printf("  %s\n", truncate(r.Text, 500))
			}
		}
	case protocol.KindUnknown:
		p.endStream()
		p.printf("● unparsed line (%d bytes)\n", len(m.Raw))
	}
}

func (p *printer) endStream() {
	if p.streaming {
		p.printf("\n")
		p.streaming = false
	}
}

func short(id domain.ApprovalID) string {
	if len(id) > 8 {
		return string(id[:8])
	}
	return string(id)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
