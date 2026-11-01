package claudecli_test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
)

// The test binary doubles as a fake `claude`: when FAKE_CLAUDE=1 it speaks stream-json on stdio.
// Binary test đóng vai `claude` giả: khi FAKE_CLAUDE=1 nó nói stream-json qua stdio.
func init() {
	if os.Getenv("FAKE_CLAUDE") == "1" {
		os.Exit(fakeClaude())
	}
}

func fakeClaude() int {
	sid := ""
	for i, a := range os.Args {
		if (a == "--session-id" || a == "--resume") && i+1 < len(os.Args) {
			sid = os.Args[i+1]
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
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)

	initDone := false
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		var msg struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		}
		if err := json.Unmarshal(sc.Bytes(), &msg); err != nil {
			fmt.Fprintln(os.Stderr, "bad input:", err)
			return 2
		}
		if !initDone {
			emit(map[string]any{"type": "system", "subtype": "init", "model": "fake", "tools": []string{"Bash"}})
			initDone = true
		}
		p := msg.Message.Content
		switch {
		case p == "env":
			result(strings.Join(os.Environ(), "\n"), false)
		case p == "args":
			result(strings.Join(os.Args[1:], "\n"), false)
		case strings.HasPrefix(p, "sleep "):
			n, _ := strconv.Atoi(strings.TrimPrefix(p, "sleep "))
			emit(map[string]any{"type": "assistant", "message": map[string]any{"role": "assistant", "content": []map[string]any{{"type": "tool_use", "id": "t1", "name": "Bash", "input": map[string]string{"command": p}}}}})
			select {
			case <-sigint:
				result("interrupted", true)
			case <-time.After(time.Duration(n) * time.Second):
				result("slept", false)
			}
		case p == "garbage":
			fmt.Println("this is not json")
			result("after garbage", false)
		case p == "crash":
			return 3
		case p == "authfail":
			emit(map[string]any{"type": "system", "subtype": "api_retry", "attempt": 1, "max_retries": 10, "error_status": 401, "error": "authentication_failed"})
			select {}
		case p == "silent":
			select {}
		default:
			emit(map[string]any{"type": "stream_event", "event": map[string]any{"type": "content_block_delta", "delta": map[string]any{"type": "text_delta", "text": "echo"}}})
			emit(map[string]any{"type": "assistant", "message": map[string]any{"role": "assistant", "content": []map[string]any{{"type": "text", "text": "echo: " + p}}}})
			result("echo: "+p, false)
		}
	}
	return 0
}
