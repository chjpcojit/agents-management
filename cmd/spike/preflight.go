package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/chjpcojit/agents-management/internal/adapter/runtime/claudecli"
)

// checkStatus is the outcome of one preflight check. / checkStatus là kết quả một bước preflight.
type checkStatus string

const (
	checkOK   checkStatus = "OK"
	checkWarn checkStatus = "WARN"
	checkFail checkStatus = "FAIL"
)

type check struct {
	Name   string
	Status checkStatus
	Detail string
}

// envCredentialKeys are variables that would switch claude away from the Max login.
// The runner strips all of them; preflight only warns so the owner knows why.
//
// envCredentialKeys là các biến làm claude không dùng login Max nữa.
// Runner loại bỏ tất cả; preflight chỉ cảnh báo để owner biết lý do.
var envCredentialKeys = []string{
	"ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN", "CLAUDE_CODE_OAUTH_TOKEN",
	"CLAUDE_CODE_USE_BEDROCK", "CLAUDE_CODE_USE_VERTEX", "CLAUDE_CODE_USE_FOUNDRY", "ANTHROPIC_BASE_URL",
}

func cmdPreflight(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("preflight", flag.ContinueOnError)
	bin := fs.String("claude", "claude", "claude binary name or path / tên hoặc đường dẫn binary claude")
	if err := fs.Parse(args); err != nil {
		return err
	}
	checks := runPreflight(context.Background(), *bin)
	failed := false
	for _, c := range checks {
		fmt.Fprintf(stdout, "[%-4s] %-22s %s\n", c.Status, c.Name, c.Detail)
		if c.Status == checkFail {
			failed = true
		}
	}
	if failed {
		return errors.New("preflight failed: fix the FAIL lines above, then run `make spike-preflight` again")
	}
	fmt.Fprintln(stdout, "\npreflight passed: run `make spike-test` (automated T1–T15) or `make spike` (interactive)")
	return nil
}

func runPreflight(ctx context.Context, bin string) []check {
	var out []check
	add := func(name string, st checkStatus, format string, a ...any) {
		out = append(out, check{Name: name, Status: st, Detail: fmt.Sprintf(format, a...)})
	}

	add("go", checkOK, "%s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		add("os", checkWarn, "Interrupt (SIGINT, T12) is not supported on Windows; use macOS, Linux or WSL")
	}

	path, err := exec.LookPath(bin)
	if err != nil {
		add("claude binary", checkFail, "%q not found in PATH: install Claude Code (https://code.claude.com/docs/en/setup)", bin)
		return out
	}
	add("claude binary", checkOK, "%s", path)

	version, err := runClaude(ctx, path, 15*time.Second, "--version")
	if err != nil {
		add("claude --version", checkFail, "%v", err)
		return out
	}
	add("claude --version", checkOK, "%s", firstLine(version))

	out = append(out, authCheck(ctx, path))

	var set []string
	for _, k := range envCredentialKeys {
		if _, ok := os.LookupEnv(k); ok {
			set = append(set, k)
		}
	}
	if len(set) > 0 {
		add("credential env", checkWarn, "%s set in this shell; the spike strips it so agents use the Max login, but plain `claude` here would not", strings.Join(set, ", "))
	} else {
		add("credential env", checkOK, "no API key or provider override in env")
	}

	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		add("loopback listen", checkFail, "cannot bind 127.0.0.1: %v", err)
	} else {
		_ = ln.Close()
		add("loopback listen", checkOK, "127.0.0.1 available for the MCP endpoint")
	}

	args, err := claudecli.BuildArgs(claudecli.Config{AgentID: "preflight", SessionID: "00000000-0000-4000-8000-000000000000", MCPURL: "http://127.0.0.1:0/mcp/preflight", MCPToken: "redacted"})
	if err == nil {
		add("flags", checkOK, "claude %s", strings.Join(redactArgs(args), " "))
	}
	return out
}

// authCheck runs `claude auth status` with the same scrubbed env the agents get, and prints
// only the login method, never the raw output (it can carry account details).
//
// authCheck chạy `claude auth status` với cùng env đã lọc như agent, và chỉ in phương thức login,
// không in output gốc (có thể chứa thông tin tài khoản).
func authCheck(ctx context.Context, path string) check {
	raw, err := runClaude(ctx, path, 20*time.Second, "auth", "status")
	if err != nil {
		return check{Name: "claude auth status", Status: checkWarn, Detail: fmt.Sprintf("could not run (%v); run `claude`, then `/login` with your Max account, and check `/status`", err)}
	}
	var st map[string]any
	if json.Unmarshal(bytes.TrimSpace(raw), &st) != nil {
		return check{Name: "claude auth status", Status: checkWarn, Detail: "output is not JSON; check `claude auth status` by hand: the method must be your claude.ai (Max) login"}
	}
	if v, ok := st["loggedIn"].(bool); ok && !v {
		return check{Name: "claude auth status", Status: checkFail, Detail: "not logged in: run `claude`, then `/login` with your Max account"}
	}
	method, _ := st["authMethod"].(string)
	switch method {
	case "claude.ai", "oauth_token":
		detail := "authMethod=" + method
		if sub, ok := st["subscriptionType"].(string); ok && sub != "" {
			detail += ", subscription=" + sub
		}
		if method == "oauth_token" {
			return check{Name: "claude auth status", Status: checkWarn, Detail: detail + "; a long-lived token, not `/login`: fine only if it belongs to your Max account (the spike never passes CLAUDE_CODE_OAUTH_TOKEN from env)"}
		}
		return check{Name: "claude auth status", Status: checkOK, Detail: detail}
	case "":
		return check{Name: "claude auth status", Status: checkWarn, Detail: "no authMethod field; check `claude auth status` by hand"}
	default:
		return check{Name: "claude auth status", Status: checkFail, Detail: "authMethod=" + method + ": the spike needs the Max login (`/login`), not an API key or cloud provider"}
	}
}

// runClaude runs claude with the agent env allowlist and a timeout. stderr is discarded.
// runClaude chạy claude với env allowlist của agent và có timeout. stderr bị bỏ qua.
func runClaude(ctx context.Context, path string, timeout time.Duration, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, args...) // #nosec G204 -- fixed args, binary from LookPath / args cố định, binary từ LookPath
	cmd.Env = claudecli.AllowedEnv(os.Environ(), nil)
	cmd.Stdin = nil
	out, err := cmd.Output()
	if ctx.Err() != nil {
		return nil, fmt.Errorf("timed out after %s", timeout)
	}
	if err != nil {
		return nil, fmt.Errorf("exit: %w", err)
	}
	return out, nil
}

// redactArgs hides the bearer token inside --mcp-config for display.
// redactArgs che bearer token trong --mcp-config khi hiển thị.
func redactArgs(args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		if strings.HasPrefix(a, "{") {
			a = "'<json>'"
		}
		out[i] = a
	}
	return out
}

func firstLine(b []byte) string {
	s := strings.TrimSpace(string(b))
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return s
}
