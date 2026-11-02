package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The test binary doubles as a fake `claude` when it is invoked through a symlink named "claude".
// The runner scrubs env, so the binary name is the trigger, not an env var.
// It speaks stream-json and calls the real MCP endpoint from --mcp-config, so the harness is
// exercised end to end without a Claude login.
//
// Binary test đóng vai `claude` giả khi được gọi qua symlink tên "claude".
// Runner lọc env nên tên binary là tín hiệu, không phải biến env.
// Nó nói stream-json và gọi endpoint MCP thật từ --mcp-config, nên harness được kiểm tra
// end to end mà không cần login Claude.
func init() {
	if filepath.Base(os.Args[0]) == "claude" {
		os.Exit(fakeClaude(os.Args[1:]))
	}
}

var (
	reBash = regexp.MustCompile(`with the Bash tool, once, and nothing else: ([^\n]+)`)
	reSend = regexp.MustCompile(`send_message once with to="([^"]+)" and text="([^"]+)"`)
)

func fakeClaude(args []string) int {
	if len(args) > 0 && args[0] == "--version" {
		fmt.Println("9.9.9 (Fake Claude Code)")
		return 0
	}
	if len(args) > 1 && args[0] == "auth" && args[1] == "status" {
		fmt.Println(`{"loggedIn":true,"authMethod":"claude.ai","subscriptionType":"max"}`)
		return 0
	}
	var sid, mcpURL, token string
	for i := 0; i+1 < len(args); i++ {
		switch args[i] {
		case "--session-id", "--resume":
			sid = args[i+1]
		case "--mcp-config":
			var cfg struct {
				MCPServers map[string]struct {
					URL     string            `json:"url"`
					Headers map[string]string `json:"headers"`
				} `json:"mcpServers"`
			}
			if err := json.Unmarshal([]byte(args[i+1]), &cfg); err == nil {
				s := cfg.MCPServers["orchestrator"]
				mcpURL, token = s.URL, strings.TrimPrefix(s.Headers["Authorization"], "Bearer ")
			}
		}
	}
	out := json.NewEncoder(os.Stdout)
	emit := func(v map[string]any) { v["session_id"] = sid; _ = out.Encode(v) }
	result := func(text string, isErr bool) {
		sub := "success"
		if isErr {
			sub = "error_during_execution"
		}
		emit(map[string]any{"type": "result", "subtype": sub, "is_error": isErr, "result": text, "num_turns": 1, "total_cost_usd": 0.001})
	}
	assistant := func(blocks ...map[string]any) {
		emit(map[string]any{"type": "assistant", "message": map[string]any{"role": "assistant", "content": blocks}})
	}
	toolResult := func(id, text string, isErr bool) {
		emit(map[string]any{"type": "user", "message": map[string]any{"role": "user", "content": []map[string]any{{"type": "tool_result", "tool_use_id": id, "content": text, "is_error": isErr}}}})
	}
	delta := func(t string) {
		emit(map[string]any{"type": "stream_event", "event": map[string]any{"type": "content_block_delta", "delta": map[string]any{"type": "text_delta", "text": t}}})
	}

	ctx := context.Background()
	var cs *mcp.ClientSession
	if mcpURL != "" {
		client := mcp.NewClient(&mcp.Implementation{Name: "fake-claude", Version: "0"}, nil)
		var err error
		cs, err = client.Connect(ctx, &mcp.StreamableClientTransport{
			Endpoint:   mcpURL,
			HTTPClient: &http.Client{Transport: bearer{token: token, base: http.DefaultTransport}},
		}, nil)
		if err != nil {
			cs = nil
		}
	}
	callText := func(name string, a map[string]any) (string, bool) {
		if cs == nil {
			return "mcp not connected", true
		}
		res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: a})
		if err != nil {
			return err.Error(), true
		}
		var b strings.Builder
		for _, c := range res.Content {
			if t, ok := c.(*mcp.TextContent); ok {
				b.WriteString(t.Text)
			}
		}
		return b.String(), res.IsError
	}

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)
	historyFile := filepath.Join(".", ".fake-history-"+sid)

	initDone := false
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		var msg struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		}
		if err := json.Unmarshal(sc.Bytes(), &msg); err != nil {
			return 2
		}
		if !initDone {
			src := "none"
			if _, ok := os.LookupEnv("ANTHROPIC_API_KEY"); ok {
				src = "ANTHROPIC_API_KEY"
			}
			emit(map[string]any{"type": "system", "subtype": "init", "model": "fake", "tools": []string{"Bash", "Read"},
				"permissionMode": "default", "apiKeySource": src,
				"mcp_servers": []map[string]string{{"name": "orchestrator", "status": map[bool]string{true: "connected", false: "failed"}[cs != nil]}}})
			initDone = true
		}
		p := msg.Message.Content
		prev, _ := os.ReadFile(historyFile) // #nosec G304 -- fake writes its own file in cwd / fake tự ghi file của nó trong cwd
		_ = os.WriteFile(historyFile, append(prev, []byte(p+"\n")...), 0o600)

		switch {
		case p == "/compact":
			emit(map[string]any{"type": "system", "subtype": "compact_boundary", "compact_metadata": map[string]any{"trigger": "manual", "pre_tokens": 1234}})
			result("", false)
		case strings.HasPrefix(p, "Write the numbers"):
			delta("1\n")
			time.Sleep(time.Second)
			result("1..40", false)
		case strings.HasPrefix(p, "Use the Read tool"):
			delta("reading")
			assistant(map[string]any{"type": "tool_use", "id": "toolu_read", "name": "Read", "input": map[string]string{"file_path": "hello.txt"}})
			b, _ := os.ReadFile("hello.txt")
			toolResult("toolu_read", string(b), false)
			result(strings.SplitN(string(b), "\n", 2)[0], false)
		case reBash.MatchString(p):
			command := strings.TrimSpace(reBash.FindStringSubmatch(p)[1])
			assistant(map[string]any{"type": "tool_use", "id": "toolu_bash", "name": "Bash", "input": map[string]string{"command": command}})
			text, _ := callText("approval_prompt", map[string]any{"tool_name": "Bash", "input": map[string]any{"command": command}, "tool_use_id": "toolu_bash"})
			var decision struct {
				Behavior string `json:"behavior"`
				Message  string `json:"message"`
			}
			_ = json.Unmarshal([]byte(text), &decision)
			if decision.Behavior != "allow" {
				toolResult("toolu_bash", decision.Message, true)
				result("I was denied: "+decision.Message, false)
				continue
			}
			interrupted := false
			switch {
			case strings.HasPrefix(command, "touch "):
				_ = os.WriteFile(strings.TrimPrefix(command, "touch "), nil, 0o600)
			case strings.HasPrefix(command, "ping -c "):
				n := strings.Fields(strings.TrimPrefix(command, "ping -c "))[0]
				d, _ := time.ParseDuration(n + "s")
				select {
				case <-sigint:
					interrupted = true
				case <-time.After(d):
				}
			}
			if interrupted {
				toolResult("toolu_bash", "interrupted", true)
				result("interrupted", true)
				continue
			}
			toolResult("toolu_bash", "ok", false)
			if m := reSend.FindStringSubmatch(p); m != nil {
				_, _ = callText("send_message", map[string]any{"to": m[1], "text": m[2]})
			}
			result("ran "+command, false)
		case reSend.MatchString(p):
			m := reSend.FindStringSubmatch(p)
			text, isErr := callText("send_message", map[string]any{"to": m[1], "text": m[2]})
			toolResult("toolu_send", text, isErr)
			result("DONE", false)
		default:
			delta("echo")
			history, _ := os.ReadFile(historyFile) // #nosec G304 -- same file as above / cùng file ở trên
			text := "echo: " + p + " | history: " + strings.ReplaceAll(string(history), "\n", " ")
			assistant(map[string]any{"type": "text", "text": text})
			result(text, false)
		}
	}
	return 0
}
