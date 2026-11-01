package protocol_test

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/chjpcojit/agents-management/internal/adapter/runtime/claudecli/protocol"
)

func TestDecodeGolden(t *testing.T) {
	f, err := os.Open("testdata/turn.ndjson")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	var got []protocol.Message
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		got = append(got, protocol.Decode(sc.Bytes()))
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}

	wantKinds := []protocol.Kind{
		protocol.KindSystemInit, protocol.KindStreamEvent, protocol.KindAssistant, protocol.KindUser,
		protocol.KindAPIRetry, protocol.KindCompactBoundary, protocol.KindSystemOther,
		protocol.KindUnknown, protocol.KindUnknown, protocol.KindResult,
	}
	if len(got) != len(wantKinds) {
		t.Fatalf("want %d lines, got %d", len(wantKinds), len(got))
	}
	for i, k := range wantKinds {
		if got[i].Kind != k {
			t.Errorf("line %d: want kind %s, got %s", i+1, k, got[i].Kind)
		}
		if len(got[i].Raw) == 0 {
			t.Errorf("line %d: raw must be kept", i+1)
		}
	}

	if in := got[0].Init; in == nil || in.Model != "claude-sonnet-5" || in.MCPServers[0].Status != "connected" {
		t.Errorf("init not decoded: %+v", in)
	}
	if got[1].TextDelta != "Hel" {
		t.Errorf("text delta: %q", got[1].TextDelta)
	}
	if b := got[2].Blocks; len(b) != 2 || b[1].Name != "Bash" || b[1].ID != "toolu_1" {
		t.Errorf("assistant blocks: %+v", b)
	}
	if b := got[3].Blocks; len(b) != 1 || b[0].ToolUseID != "toolu_1" {
		t.Errorf("tool result blocks: %+v", b)
	}
	if r := got[4].Retry; r == nil || r.Error != "overloaded" || r.ErrorStatus == nil || *r.ErrorStatus != 529 {
		t.Errorf("retry: %+v", r)
	}
	if got[8].DecodeError == nil {
		t.Error("invalid JSON must set DecodeError")
	}
	if r := got[9].Result; r == nil || r.Text != "Hello" || r.TotalCostUSD != 0.0123 || got[9].SessionID == "" {
		t.Errorf("result: %+v", r)
	}
}

func TestDecodeStringContent(t *testing.T) {
	m := protocol.Decode([]byte(`{"type":"user","message":{"role":"user","content":"hi"}}`))
	if len(m.Blocks) != 1 || m.Blocks[0].Text != "hi" {
		t.Fatalf("got %+v", m.Blocks)
	}
}

func TestEncodeUserMessage(t *testing.T) {
	line, err := protocol.EncodeUserMessage("hello \"world\"\nsecond line")
	if err != nil {
		t.Fatal(err)
	}
	if line[len(line)-1] != '\n' {
		t.Fatal("must end with newline")
	}
	var v struct {
		Type    string `json:"type"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		Parent *string `json:"parent_tool_use_id"`
	}
	if err := json.Unmarshal(line, &v); err != nil {
		t.Fatal(err)
	}
	if v.Type != "user" || v.Message.Role != "user" || v.Message.Content != "hello \"world\"\nsecond line" || v.Parent != nil {
		t.Fatalf("got %+v", v)
	}
	// Exactly one line: the embedded newline must be escaped. / Đúng một dòng: newline bên trong phải được escape.
	for _, c := range line[:len(line)-1] {
		if c == '\n' {
			t.Fatal("embedded raw newline breaks NDJSON framing")
		}
	}
	if _, err := protocol.EncodeUserMessage(""); !errors.Is(err, protocol.ErrEmptyPrompt) {
		t.Fatalf("want ErrEmptyPrompt, got %v", err)
	}
}
