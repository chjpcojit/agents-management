package main

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chjpcojit/agents-management/internal/adapter/runtime/claudecli/protocol"
)

type bearer struct {
	token string
	base  http.RoundTripper
}

func (b bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.token)
	return b.base.RoundTrip(r)
}

// fakeBin links the test binary as "claude" so init() turns it into the fake.
// fakeBin tạo symlink binary test với tên "claude" để init() biến nó thành bản giả.
func fakeBin(t *testing.T) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "claude")
	if err := os.Symlink(exe, bin); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}
	return bin
}

func TestPreflightAgainstFake(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-fake")
	checks := runPreflight(context.Background(), fakeBin(t))
	got := map[string]checkStatus{}
	for _, c := range checks {
		got[c.Name] = c.Status
	}
	for name, want := range map[string]checkStatus{
		"claude binary": checkOK, "claude --version": checkOK, "claude auth status": checkOK,
		"credential env": checkWarn, "loopback listen": checkOK, "flags": checkOK,
	} {
		if got[name] != want {
			t.Errorf("%s = %q, want %q (all: %v)", name, got[name], want, got)
		}
	}
	for _, c := range checks {
		if c.Name == "flags" && strings.Contains(c.Detail, "Bearer") {
			t.Error("flags line must not show the bearer token")
		}
	}
}

func TestPreflightMissingBinary(t *testing.T) {
	checks := runPreflight(context.Background(), filepath.Join(t.TempDir(), "nope"))
	if last := checks[len(checks)-1]; last.Status != checkFail {
		t.Fatalf("missing binary must fail, got %+v", last)
	}
}

// TestScenariosAgainstFake runs the harness end to end: real approval service, real MCP server,
// real runner, fake claude. It proves the wiring and the checks, not Claude's behaviour.
//
// TestScenariosAgainstFake chạy harness end to end: approval service thật, MCP server thật,
// runner thật, claude giả. Nó chứng minh phần nối dây và các bước kiểm tra, không phải hành vi của Claude.
func TestScenariosAgainstFake(t *testing.T) {
	env := scenarioEnv{
		ClaudeBin:  fakeBin(t),
		SandboxDir: t.TempDir(),
		T7Wait:     2 * time.Second,
		TurnLimit:  20 * time.Second,
		Log:        newLogger(false),
	}
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-parent-env-fake")
	tests := map[string]Status{
		"T1": StatusPass, "T2": StatusPass, "T3": StatusObserved, "T4": StatusPass, "T5": StatusPass,
		"T6": StatusPass, "T7": StatusPass, "T8": StatusPass, "T9": StatusObserved, "T10": StatusPass,
		"T11": StatusObserved, "T12": StatusPass, "T13": StatusPass, "T14": StatusPass, "T15": StatusPass,
	}
	var results []CaseResult
	for _, s := range scenarios() {
		want, ok := tests[s.ID]
		if !ok {
			t.Fatalf("scenario %s has no expectation", s.ID)
		}
		t.Run(s.ID, func(t *testing.T) {
			c := CaseResult{ID: s.ID, Name: s.Name, Status: StatusPass}
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			if err := runCase(ctx, s, env, &c); err != nil {
				t.Fatalf("harness error: %v", err)
			}
			if c.Status != want {
				t.Fatalf("status %s, want %s; notes:\n%s", c.Status, want, strings.Join(c.Notes, "\n"))
			}
			results = append(results, c)
		})
	}
	report := renderReport(nil, "", results)
	for _, want := range []string{"## Summary", "| T5 |", "PASS"} {
		if !strings.Contains(report, want) {
			t.Errorf("report misses %q", want)
		}
	}
}

func TestInteractiveSession(t *testing.T) {
	in := strings.NewReader("hello\n:sid\n:pending\n:bogus\n:quit\n")
	var out bytes.Buffer
	done := make(chan error, 1)
	go func() { done <- cmdInteractive([]string{"-claude", fakeBin(t), "-dir", t.TempDir()}, in, &out) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("interactive session did not exit")
	}
	for _, want := range []string{"agent started", "session=", "no pending approvals", "unknown command :bogus", "closing agent"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output misses %q:\n%s", want, out.String())
		}
	}
}

// The lines come from the owner's real run (spike T5, 2026-11-01): Bash was accepted by a user rule.
// Các dòng lấy từ lần chạy thật của owner (spike T5, 2026-11-01): Bash được rule của user chấp nhận.
func TestCheckNotBypassedCatchesUserRule(t *testing.T) {
	use := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_014p","name":"Bash","input":{"command":"touch x"}}]}}`
	res := `{"type":"user","message":{"role":"user","content":[{"tool_use_id":"toolu_014p","type":"tool_result","content":"(Bash completed with no output)","is_error":false}]},"tool_result_meta":[{"id":"toolu_014p","permission_decision":{"decision":"accept","source":"user_permanent","reason_type":"rule"}}]}`
	msgs := []timedMessage{{Msg: protocol.Decode([]byte(use))}, {Msg: protocol.Decode([]byte(res))}}

	c := CaseResult{Status: StatusPass}
	checkNotBypassed(&c, msgs, "Bash")
	if c.Status != StatusFail {
		t.Fatalf("a user rule deciding Bash must fail the case, notes: %v", c.Notes)
	}

	c = CaseResult{Status: StatusPass}
	checkNotBypassed(&c, msgs, "Edit")
	if c.Status != StatusPass {
		t.Fatalf("other tools are not affected, notes: %v", c.Notes)
	}
}
