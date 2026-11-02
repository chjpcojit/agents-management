// Package mcpbus is the MCP delivery layer agents connect to: the permission prompt tool that feeds the
// approval inbox, and the (stub) office bus tools. Each agent has its own URL and bearer token.
//
// Package mcpbus là delivery layer MCP mà agent kết nối vào: permission prompt tool đẩy yêu cầu vào
// approval inbox, và các tool office bus (stub). Mỗi agent có URL và bearer token riêng.
package mcpbus

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/chjpcojit/agents-management/internal/domain"
	"github.com/chjpcojit/agents-management/internal/usecase/approval"
)

// Tool names as registered on the server. Claude sees them as mcp__orchestrator__<name>.
// Tên tool đăng ký trên server. Claude thấy chúng dưới dạng mcp__orchestrator__<name>.
const (
	ToolApprovalPrompt = "approval_prompt"
	ToolSendMessage    = "send_message"
)

// PathPrefix is where Handler must be mounted. / PathPrefix là nơi phải mount Handler.
const PathPrefix = "/mcp/"

// Approvals is the slice of the approval use case this layer needs.
// Approvals là phần của use case approval mà layer này cần.
type Approvals interface {
	Request(ctx context.Context, in approval.RequestInput) (domain.Approval, error)
	AwaitDecision(ctx context.Context, id domain.ApprovalID) (domain.Approval, error)
	Abandon(ctx context.Context, id domain.ApprovalID, reason string) (domain.Approval, error)
}

// MessageSink receives office bus messages. Stub until the office-bus feature exists.
// MessageSink nhận message của office bus. Là stub cho tới khi có feature office-bus.
type MessageSink func(ctx context.Context, from domain.AgentID, to, text string) error

// Hooks let the spike harness observe traffic without touching business logic.
// Hooks cho phép spike harness quan sát traffic mà không động vào business logic.
type Hooks struct {
	// OnApprovalCall receives the agent and the top-level argument keys of every approval_prompt call (Q-2).
	// OnApprovalCall nhận agent và các key cấp cao nhất của mỗi lần gọi approval_prompt (Q-2).
	OnApprovalCall func(agent domain.AgentID, argKeys []string)
	// OnApprovalOpened fires after an approval is stored. / OnApprovalOpened chạy sau khi approval được lưu.
	OnApprovalOpened func(a domain.Approval)
}

// Server serves MCP for every registered agent.
// Server phục vụ MCP cho mọi agent đã đăng ký.
type Server struct {
	approvals Approvals
	sink      MessageSink
	hooks     Hooks
	log       *slog.Logger

	mu      sync.RWMutex
	tokens  map[domain.AgentID][]byte
	servers map[domain.AgentID]*mcp.Server

	handler http.Handler
}

// New builds a Server. sink may be nil, in which case send_message reports the bus as unavailable.
// New tạo Server. sink có thể nil, khi đó send_message báo bus chưa sẵn sàng.
func New(approvals Approvals, sink MessageSink, hooks Hooks, log *slog.Logger) (*Server, error) {
	if approvals == nil {
		return nil, fmt.Errorf("%w: approvals is required", domain.ErrInvalidArgument)
	}
	if log == nil {
		log = slog.Default()
	}
	s := &Server{
		approvals: approvals,
		sink:      sink,
		hooks:     hooks,
		log:       log,
		tokens:    make(map[domain.AgentID][]byte),
		servers:   make(map[domain.AgentID]*mcp.Server),
	}
	s.handler = s.authenticate(mcp.NewStreamableHTTPHandler(s.serverFor, &mcp.StreamableHTTPOptions{Logger: log}))
	return s, nil
}

// Register creates a fresh bearer token for agent and returns it. Re-registering rotates the token.
// Register tạo bearer token mới cho agent và trả về. Đăng ký lại sẽ xoay vòng token.
func (s *Server) Register(agent domain.AgentID) (string, error) {
	if strings.TrimSpace(string(agent)) == "" || strings.ContainsAny(string(agent), "/?#") {
		return "", fmt.Errorf("%w: agent id %q is not usable in a URL path", domain.ErrInvalidArgument, agent)
	}
	var b [32]byte
	_, _ = rand.Read(b[:])
	token := hex.EncodeToString(b[:])
	s.mu.Lock()
	s.tokens[agent] = []byte(token)
	s.mu.Unlock()
	return token, nil
}

// Unregister removes the agent; its token stops working immediately.
// Unregister xóa agent; token của nó hết hiệu lực ngay.
func (s *Server) Unregister(agent domain.AgentID) {
	s.mu.Lock()
	delete(s.tokens, agent)
	delete(s.servers, agent)
	s.mu.Unlock()
}

// Handler serves every path under PathPrefix. / Handler phục vụ mọi path dưới PathPrefix.
func (s *Server) Handler() http.Handler { return s.handler }

// AgentURL returns the MCP endpoint of agent below baseURL (e.g. http://127.0.0.1:8788).
// AgentURL trả về endpoint MCP của agent dưới baseURL (ví dụ http://127.0.0.1:8788).
func AgentURL(baseURL string, agent domain.AgentID) string {
	return strings.TrimRight(baseURL, "/") + PathPrefix + string(agent)
}

type agentKey struct{}

// authenticate checks the bearer token on every request, not only at session start,
// so a leaked session id alone is not enough to act as an agent.
//
// authenticate kiểm tra bearer token ở mọi request chứ không chỉ lúc mở session,
// để chỉ lộ session id thôi thì không thể giả danh agent.
func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		agent, ok := agentFromPath(r.URL.Path)
		if !ok {
			http.NotFound(w, r)
			return
		}
		got, hasBearer := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		s.mu.RLock()
		want, known := s.tokens[agent]
		s.mu.RUnlock()
		if !hasBearer || !known || subtle.ConstantTimeCompare([]byte(got), want) != 1 {
			s.log.Warn("mcp request rejected", slog.String("agent_id", string(agent)), slog.Bool("known_agent", known))
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), agentKey{}, agent)))
	})
}

func agentFromPath(p string) (domain.AgentID, bool) {
	rest, ok := strings.CutPrefix(p, PathPrefix)
	if !ok {
		return "", false
	}
	id, _, _ := strings.Cut(rest, "/")
	if id == "" {
		return "", false
	}
	return domain.AgentID(id), true
}

// serverFor returns the MCP server bound to the authenticated agent, so tools always know who calls them.
// serverFor trả về MCP server gắn với agent đã xác thực, để tool luôn biết ai đang gọi.
func (s *Server) serverFor(r *http.Request) *mcp.Server {
	agent, ok := r.Context().Value(agentKey{}).(domain.AgentID)
	if !ok {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if srv, ok := s.servers[agent]; ok {
		return srv
	}
	srv := mcp.NewServer(&mcp.Implementation{Name: "orchestrator", Version: "0.1.0"}, nil)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        ToolApprovalPrompt,
		Description: "Permission prompt handler: asks the human owner to allow or deny a tool call.",
	}, s.approvalPrompt(agent))
	mcp.AddTool(srv, &mcp.Tool{
		Name:        ToolSendMessage,
		Description: "Send a short message to another agent or to the shared channel.",
	}, s.sendMessage(agent))
	s.servers[agent] = srv
	return srv
}

// permissionResult is what Claude Code expects back from the permission prompt tool, as JSON text.
// ⚠️ Shape not documented yet (Q-2): verified by spike T5/T6.
//
// permissionResult là thứ Claude Code chờ nhận lại từ permission prompt tool, dưới dạng JSON text.
// ⚠️ Shape chưa có docs (Q-2): được kiểm chứng ở spike T5/T6.
type permissionResult struct {
	Behavior     string         `json:"behavior"`
	UpdatedInput map[string]any `json:"updatedInput,omitempty"`
	Message      string         `json:"message,omitempty"`
}

func (s *Server) approvalPrompt(agent domain.AgentID) mcp.ToolHandlerFor[map[string]any, any] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
		if s.hooks.OnApprovalCall != nil {
			keys := make([]string, 0, len(args))
			for k := range args {
				keys = append(keys, k)
			}
			slices.Sort(keys)
			s.hooks.OnApprovalCall(agent, keys)
		}
		toolName, _ := args["tool_name"].(string)
		toolInput, _ := args["input"].(map[string]any)
		rawInput, err := json.Marshal(toolInput)
		if err != nil {
			return deny("tool input could not be encoded"), nil, nil
		}

		a, err := s.approvals.Request(ctx, approval.RequestInput{
			AgentID:  agent,
			Kind:     domain.ApprovalKindTool,
			ToolName: toolName,
			Input:    rawInput,
			Risk:     domain.ClassifyRisk(toolName),
		})
		if err != nil {
			s.log.WarnContext(ctx, "approval request rejected", slog.String("agent_id", string(agent)), slog.String("tool", toolName), slog.Any("error", err))
			return deny("approval request was invalid"), nil, nil
		}
		if s.hooks.OnApprovalOpened != nil {
			s.hooks.OnApprovalOpened(a)
		}

		decided, err := s.approvals.AwaitDecision(ctx, a.ID)
		switch {
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			// Claude Code gave up on the call (its tool timeout, spike T7) or the agent went away.
			// End the approval now so the inbox never shows a request nobody is waiting on.
			// Claude Code đã bỏ lệnh gọi (timeout tool, spike T7) hoặc agent đã đi. Kết thúc approval
			// ngay để inbox không hiện yêu cầu mà không còn ai chờ.
			actx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			if _, aerr := s.approvals.Abandon(actx, a.ID, "agent stopped waiting, denied by default"); aerr != nil {
				s.log.WarnContext(actx, "abandon approval failed", slog.String("approval_id", string(a.ID)), slog.Any("error", aerr))
			}
			cancel()
			s.log.InfoContext(ctx, "approval wait cancelled", slog.String("approval_id", string(a.ID)))
			return deny("approval wait was cancelled"), nil, nil
		case err != nil:
			s.log.ErrorContext(ctx, "approval wait failed", slog.String("approval_id", string(a.ID)), slog.Any("error", err))
			return deny("approval could not be completed"), nil, nil
		}

		switch decided.Status {
		case domain.ApprovalAllowed:
			return textResult(permissionResult{Behavior: "allow", UpdatedInput: nonNil(toolInput)}), nil, nil
		case domain.ApprovalExpired:
			return deny("no decision before the deadline; denied by default"), nil, nil
		default:
			msg := "denied by the owner"
			if decided.Reason != "" {
				msg += ": " + decided.Reason
			}
			return deny(msg), nil, nil
		}
	}
}

type sendMessageArgs struct {
	To   string `json:"to" jsonschema:"agent id or channel name"`
	Text string `json:"text" jsonschema:"message text, at most 4000 characters"`
}

func (s *Server) sendMessage(agent domain.AgentID) mcp.ToolHandlerFor[sendMessageArgs, any] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in sendMessageArgs) (*mcp.CallToolResult, any, error) {
		switch {
		case strings.TrimSpace(in.To) == "" || strings.TrimSpace(in.Text) == "":
			return errorResult("to and text are required"), nil, nil
		case len(in.Text) > 4000:
			return errorResult("text is longer than 4000 characters"), nil, nil
		case s.sink == nil:
			return errorResult("office bus is not available yet"), nil, nil
		}
		if err := s.sink(ctx, agent, in.To, in.Text); err != nil {
			s.log.WarnContext(ctx, "bus message rejected", slog.String("agent_id", string(agent)), slog.Any("error", err))
			return errorResult("message was not delivered"), nil, nil
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "queued"}}}, nil, nil
	}
}

func deny(msg string) *mcp.CallToolResult {
	return textResult(permissionResult{Behavior: "deny", Message: msg})
}

func textResult(v permissionResult) *mcp.CallToolResult {
	b, _ := json.Marshal(v) // fixed struct of strings and maps from JSON; cannot fail / struct cố định, không thể lỗi
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: msg}}}
}

func nonNil(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	return m
}
