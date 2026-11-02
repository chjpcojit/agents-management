package domain

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ApprovalID identifies one approval request.
// ApprovalID định danh một yêu cầu duyệt.
type ApprovalID string

// AgentID identifies one managed agent.
// AgentID định danh một agent được quản lý.
type AgentID string

// ApprovalKind tells whether the agent asks to run a tool or asks the user a question.
// ApprovalKind cho biết agent xin chạy tool hay chủ động hỏi user.
type ApprovalKind string

const (
	ApprovalKindTool ApprovalKind = "tool"
	ApprovalKindAsk  ApprovalKind = "ask"
)

// ApprovalStatus is the lifecycle state of an approval.
// ApprovalStatus là trạng thái vòng đời của approval.
type ApprovalStatus string

const (
	ApprovalPending ApprovalStatus = "pending"
	ApprovalAllowed ApprovalStatus = "allowed"
	ApprovalDenied  ApprovalStatus = "denied"
	ApprovalExpired ApprovalStatus = "expired"
)

// Decision is the answer a human gives to a pending approval.
// Decision là câu trả lời của người duyệt cho một approval đang pending.
type Decision string

const (
	DecisionAllow Decision = "allow"
	DecisionDeny  Decision = "deny"
)

// RiskLevel marks how dangerous the requested action is, for highlighting in the inbox.
// RiskLevel đánh dấu mức nguy hiểm của hành động, để làm nổi bật trong inbox duyệt.
type RiskLevel string

const (
	RiskLow  RiskLevel = "low"
	RiskHigh RiskLevel = "high"
)

// Approval is a request from an agent that blocks until a human decides or the deadline passes.
// Expiry always resolves to a deny (fail closed).
//
// Approval là yêu cầu từ agent, bị chặn cho tới khi người duyệt quyết định hoặc quá deadline.
// Hết hạn luôn được xử lý như deny (fail closed).
type Approval struct {
	ID        ApprovalID
	AgentID   AgentID
	SessionID string
	Kind      ApprovalKind
	ToolName  string
	Input     json.RawMessage
	Risk      RiskLevel
	Status    ApprovalStatus
	CreatedAt time.Time
	Deadline  time.Time
	DecidedBy string
	DecidedAt time.Time
	Reason    string
}

// NewApprovalParams carries the inputs needed to open an approval.
// NewApprovalParams chứa input cần để mở một approval.
type NewApprovalParams struct {
	ID        ApprovalID
	AgentID   AgentID
	SessionID string
	Kind      ApprovalKind
	ToolName  string
	Input     json.RawMessage
	Risk      RiskLevel
	Now       time.Time
	Timeout   time.Duration
}

// NewApproval validates the params and returns a pending approval.
// NewApproval kiểm tra params và trả về một approval ở trạng thái pending.
func NewApproval(p NewApprovalParams) (Approval, error) {
	var problems []string
	if strings.TrimSpace(string(p.ID)) == "" {
		problems = append(problems, "id is required")
	}
	if strings.TrimSpace(string(p.AgentID)) == "" {
		problems = append(problems, "agent id is required")
	}
	switch p.Kind {
	case ApprovalKindTool:
		if strings.TrimSpace(p.ToolName) == "" {
			problems = append(problems, "tool name is required for kind tool")
		}
	case ApprovalKindAsk:
	default:
		problems = append(problems, fmt.Sprintf("unknown kind %q", p.Kind))
	}
	switch p.Risk {
	case RiskLow, RiskHigh:
	default:
		problems = append(problems, fmt.Sprintf("unknown risk %q", p.Risk))
	}
	if p.Timeout <= 0 {
		problems = append(problems, "timeout must be positive")
	}
	if p.Now.IsZero() {
		problems = append(problems, "now is required")
	}
	if len(p.Input) > 0 && !json.Valid(p.Input) {
		problems = append(problems, "input must be valid JSON")
	}
	if len(problems) > 0 {
		return Approval{}, fmt.Errorf("%w: %s", ErrInvalidArgument, strings.Join(problems, "; "))
	}

	// Copy the input so later mutation by the caller cannot change the stored request.
	// Copy input để caller sửa slice sau đó cũng không làm đổi request đã lưu.
	input := append(json.RawMessage(nil), p.Input...)

	return Approval{
		ID:        p.ID,
		AgentID:   p.AgentID,
		SessionID: p.SessionID,
		Kind:      p.Kind,
		ToolName:  p.ToolName,
		Input:     input,
		Risk:      p.Risk,
		Status:    ApprovalPending,
		CreatedAt: p.Now.UTC(),
		Deadline:  p.Now.UTC().Add(p.Timeout),
	}, nil
}

// Decide records a human decision. It fails if the approval is no longer pending or the deadline passed.
// Decide ghi nhận quyết định của người duyệt. Lỗi nếu approval không còn pending hoặc đã quá deadline.
func (a *Approval) Decide(d Decision, by, reason string, now time.Time) error {
	if a.Status != ApprovalPending {
		return fmt.Errorf("%w: status is %s", ErrApprovalAlreadyDecided, a.Status)
	}
	if strings.TrimSpace(by) == "" {
		return fmt.Errorf("%w: decider is required", ErrInvalidArgument)
	}
	if !now.Before(a.Deadline) {
		return ErrApprovalExpired
	}
	switch d {
	case DecisionAllow:
		a.Status = ApprovalAllowed
	case DecisionDeny:
		a.Status = ApprovalDenied
	default:
		return fmt.Errorf("%w: unknown decision %q", ErrInvalidArgument, d)
	}
	a.DecidedBy = by
	a.DecidedAt = now.UTC()
	a.Reason = reason
	return nil
}

// Expire moves a pending approval past its deadline to expired. It reports whether the state changed.
// Expire chuyển approval pending đã quá deadline sang expired. Trả về true nếu trạng thái thay đổi.
func (a *Approval) Expire(now time.Time) bool {
	if a.Status != ApprovalPending || now.Before(a.Deadline) {
		return false
	}
	a.Status = ApprovalExpired
	a.DecidedBy = "system"
	a.DecidedAt = now.UTC()
	a.Reason = "deadline passed, denied by default"
	return true
}

// Abandon ends a pending approval whose requester stopped waiting (the agent timed out or exited).
// It resolves as expired, i.e. deny, so a later decision cannot unblock anything. It reports whether the state changed.
//
// Abandon kết thúc approval pending khi bên yêu cầu đã bỏ chờ (agent timeout hoặc đã thoát).
// Kết quả là expired, tức deny, nên quyết định đến sau không mở khóa được gì. Trả về true nếu trạng thái thay đổi.
func (a *Approval) Abandon(now time.Time, reason string) bool {
	if a.Status != ApprovalPending {
		return false
	}
	if strings.TrimSpace(reason) == "" {
		reason = "requester stopped waiting, denied by default"
	}
	a.Status = ApprovalExpired
	a.DecidedBy = "system"
	a.DecidedAt = now.UTC()
	a.Reason = reason
	return true
}

// IsAllowed reports whether the agent may proceed. Every status except allowed means no.
// IsAllowed cho biết agent có được tiếp tục không. Mọi trạng thái trừ allowed đều là không.
func (a Approval) IsAllowed() bool {
	return a.Status == ApprovalAllowed
}

// IsTerminal reports whether the approval can no longer change.
// IsTerminal cho biết approval đã ở trạng thái cuối, không thể thay đổi nữa.
func (a Approval) IsTerminal() bool {
	return a.Status != ApprovalPending
}

// ClassifyRisk marks tools that change the machine or reach the network as high risk.
// Unknown tools, including MCP tools, default to high: fail closed.
//
// ClassifyRisk đánh dấu tool thay đổi máy hoặc truy cập mạng là rủi ro cao.
// Tool lạ, kể cả MCP tool, mặc định là cao: fail closed.
func ClassifyRisk(toolName string) RiskLevel {
	switch toolName {
	case "Read", "Glob", "Grep", "LS", "TodoWrite", "TaskCreate", "TaskUpdate", "TaskList", "TaskGet":
		return RiskLow
	default:
		return RiskHigh
	}
}
