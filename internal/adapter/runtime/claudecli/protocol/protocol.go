// Package protocol decodes and encodes the stream-json lines exchanged with `claude -p`.
// Decoding is lenient: unknown types and fields are kept raw so a newer CLI never breaks the stream.
//
// Package protocol decode và encode các dòng stream-json trao đổi với `claude -p`.
// Decode theo kiểu dễ dãi: type và field lạ được giữ nguyên dạng thô để bản CLI mới không làm gãy stream.
package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// Kind classifies a decoded line.
// Kind phân loại một dòng đã decode.
type Kind string

const (
	KindSystemInit      Kind = "system/init"
	KindSystemOther     Kind = "system/other"
	KindCompactBoundary Kind = "system/compact_boundary"
	KindAPIRetry        Kind = "system/api_retry"
	KindAssistant       Kind = "assistant"
	KindUser            Kind = "user"
	KindStreamEvent     Kind = "stream_event"
	KindResult          Kind = "result"
	KindUnknown         Kind = "unknown"
)

// Message is one decoded stdout line. Raw always holds the original bytes.
// Message là một dòng stdout đã decode. Raw luôn giữ nguyên byte gốc.
type Message struct {
	Kind            Kind
	Type            string
	Subtype         string
	SessionID       string
	UUID            string
	ParentToolUseID string
	Raw             json.RawMessage

	// Populated for KindAssistant and KindUser. / Có giá trị với KindAssistant và KindUser.
	Blocks []Block
	// Populated for KindResult. / Có giá trị với KindResult.
	Result *Result
	// Populated for KindSystemInit. / Có giá trị với KindSystemInit.
	Init *Init
	// Populated for KindAPIRetry. / Có giá trị với KindAPIRetry.
	Retry *APIRetry
	// Populated for KindStreamEvent when the event is a text delta. / Có giá trị khi stream_event là text delta.
	TextDelta string
	// DecodeError is set when the line is not valid JSON. / Có giá trị khi dòng không phải JSON hợp lệ.
	DecodeError error
}

// Block is one content block of an assistant or user message.
// Block là một content block trong message của assistant hoặc user.
type Block struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

// Result is the final message of a turn.
// Result là message cuối cùng của một turn.
type Result struct {
	Subtype      string          `json:"subtype"`
	IsError      bool            `json:"is_error"`
	Text         string          `json:"result"`
	NumTurns     int             `json:"num_turns"`
	DurationMS   int64           `json:"duration_ms"`
	TotalCostUSD float64         `json:"total_cost_usd"`
	Usage        json.RawMessage `json:"usage,omitempty"`
}

// Init is the session metadata emitted at the start of a session.
// Init là metadata của session, được phát khi session bắt đầu.
type Init struct {
	Model          string   `json:"model"`
	CWD            string   `json:"cwd"`
	Tools          []string `json:"tools"`
	PermissionMode string   `json:"permissionMode"`
	SlashCommands  []string `json:"slash_commands"`
	APIKeySource   string   `json:"apiKeySource"`
	MCPServers     []struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"mcp_servers"`
}

// APIRetry reports a retried API request.
// APIRetry báo một request API đang được retry.
type APIRetry struct {
	Attempt     int    `json:"attempt"`
	MaxRetries  int    `json:"max_retries"`
	ErrorStatus *int   `json:"error_status"`
	Error       string `json:"error"`
}

type envelope struct {
	Type            string          `json:"type"`
	Subtype         string          `json:"subtype"`
	SessionID       string          `json:"session_id"`
	UUID            string          `json:"uuid"`
	ParentToolUseID *string         `json:"parent_tool_use_id"`
	Message         json.RawMessage `json:"message"`
	Event           json.RawMessage `json:"event"`
}

// Decode parses one line. It never returns an error: bad input yields KindUnknown with DecodeError set.
// Decode parse một dòng. Không bao giờ trả lỗi: input hỏng cho ra KindUnknown và DecodeError.
func Decode(line []byte) Message {
	raw := append(json.RawMessage(nil), bytes.TrimSpace(line)...)
	m := Message{Kind: KindUnknown, Raw: raw}

	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		m.DecodeError = fmt.Errorf("decode stream line: %w", err)
		return m
	}
	m.Type, m.Subtype, m.SessionID, m.UUID = env.Type, env.Subtype, env.SessionID, env.UUID
	if env.ParentToolUseID != nil {
		m.ParentToolUseID = *env.ParentToolUseID
	}

	switch env.Type {
	case "system":
		m.decodeSystem(raw)
	case "assistant", "user":
		m.Kind = Kind(env.Type)
		var body struct {
			Content json.RawMessage `json:"content"`
		}
		if len(env.Message) > 0 && json.Unmarshal(env.Message, &body) == nil {
			m.Blocks = decodeContent(body.Content)
		}
	case "stream_event":
		m.Kind = KindStreamEvent
		var ev struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
		}
		if json.Unmarshal(env.Event, &ev) == nil && ev.Delta.Type == "text_delta" {
			m.TextDelta = ev.Delta.Text
		}
	case "result":
		var r Result
		if err := json.Unmarshal(raw, &r); err == nil {
			m.Kind = KindResult
			m.Result = &r
		}
	}
	return m
}

func (m *Message) decodeSystem(raw json.RawMessage) {
	switch m.Subtype {
	case "init":
		var in Init
		if json.Unmarshal(raw, &in) == nil {
			m.Kind, m.Init = KindSystemInit, &in
			return
		}
	case "compact_boundary":
		m.Kind = KindCompactBoundary
		return
	case "api_retry":
		var r APIRetry
		if json.Unmarshal(raw, &r) == nil {
			m.Kind, m.Retry = KindAPIRetry, &r
			return
		}
	}
	m.Kind = KindSystemOther
}

// decodeContent accepts both a plain string and an array of blocks.
// decodeContent nhận cả chuỗi thường lẫn mảng block.
func decodeContent(c json.RawMessage) []Block {
	if len(c) == 0 {
		return nil
	}
	var s string
	if json.Unmarshal(c, &s) == nil {
		return []Block{{Type: "text", Text: s}}
	}
	var blocks []Block
	if json.Unmarshal(c, &blocks) == nil {
		return blocks
	}
	return nil
}

// ErrEmptyPrompt is returned when the prompt to encode is empty.
// ErrEmptyPrompt trả về khi prompt cần encode bị rỗng.
var ErrEmptyPrompt = errors.New("empty prompt")

// EncodeUserMessage returns one stream-json line (with trailing newline) carrying a text prompt.
// EncodeUserMessage trả về một dòng stream-json (có newline cuối) chứa prompt dạng text.
func EncodeUserMessage(text string) ([]byte, error) {
	if text == "" {
		return nil, ErrEmptyPrompt
	}
	line, err := json.Marshal(struct {
		Type            string  `json:"type"`
		Message         any     `json:"message"`
		ParentToolUseID *string `json:"parent_tool_use_id"`
	}{
		Type:    "user",
		Message: map[string]string{"role": "user", "content": text},
	})
	if err != nil {
		return nil, fmt.Errorf("encode user message: %w", err)
	}
	return append(line, '\n'), nil
}
