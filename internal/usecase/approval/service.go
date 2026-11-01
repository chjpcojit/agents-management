// Package approval implements the central approval inbox: open a request, decide it, expire it.
// Package approval hiện thực inbox duyệt tập trung: mở yêu cầu, quyết định, cho hết hạn.
package approval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/chjpcojit/agents-management/internal/domain"
	"github.com/chjpcojit/agents-management/internal/usecase/port"
)

// Service coordinates approval use cases. Safe for concurrent use when the repository is.
// Service điều phối các use case approval. An toàn khi dùng đồng thời nếu repository an toàn.
type Service struct {
	repo    port.ApprovalRepository
	clock   port.Clock
	ids     port.IDGenerator
	timeout time.Duration
	log     *slog.Logger
}

// NewService builds a Service. timeout is the deadline given to every new approval.
// NewService tạo Service. timeout là deadline gán cho mọi approval mới.
func NewService(repo port.ApprovalRepository, clock port.Clock, ids port.IDGenerator, timeout time.Duration, log *slog.Logger) (*Service, error) {
	switch {
	case repo == nil:
		return nil, fmt.Errorf("%w: repository is required", domain.ErrInvalidArgument)
	case clock == nil:
		return nil, fmt.Errorf("%w: clock is required", domain.ErrInvalidArgument)
	case ids == nil:
		return nil, fmt.Errorf("%w: id generator is required", domain.ErrInvalidArgument)
	case timeout <= 0:
		return nil, fmt.Errorf("%w: timeout must be positive", domain.ErrInvalidArgument)
	}
	if log == nil {
		log = slog.Default()
	}
	return &Service{repo: repo, clock: clock, ids: ids, timeout: timeout, log: log}, nil
}

// RequestInput describes what an agent asks permission for.
// RequestInput mô tả agent đang xin quyền làm gì.
type RequestInput struct {
	AgentID   domain.AgentID
	SessionID string
	Kind      domain.ApprovalKind
	ToolName  string
	Input     json.RawMessage
	Risk      domain.RiskLevel
}

// Request opens a pending approval with the configured deadline.
// Request mở một approval pending với deadline đã cấu hình.
func (s *Service) Request(ctx context.Context, in RequestInput) (domain.Approval, error) {
	a, err := domain.NewApproval(domain.NewApprovalParams{
		ID:        domain.ApprovalID(s.ids.NewID()),
		AgentID:   in.AgentID,
		SessionID: in.SessionID,
		Kind:      in.Kind,
		ToolName:  in.ToolName,
		Input:     in.Input,
		Risk:      in.Risk,
		Now:       s.clock.Now(),
		Timeout:   s.timeout,
	})
	if err != nil {
		return domain.Approval{}, err
	}
	if err := s.repo.Create(ctx, a); err != nil {
		return domain.Approval{}, fmt.Errorf("create approval: %w", err)
	}
	// Never log the tool input: it can hold file contents or secrets.
	// Không log tool input: nó có thể chứa nội dung file hoặc secret.
	s.log.InfoContext(ctx, "approval requested",
		slog.String("approval_id", string(a.ID)),
		slog.String("agent_id", string(a.AgentID)),
		slog.String("kind", string(a.Kind)),
		slog.String("tool", a.ToolName),
		slog.String("risk", string(a.Risk)),
		slog.Time("deadline", a.Deadline),
	)
	return a, nil
}

// Decide applies a human decision. A decision arriving after the deadline expires the approval
// and returns domain.ErrApprovalExpired, so a late "allow" can never unblock the agent.
//
// Decide áp dụng quyết định của người duyệt. Quyết định đến sau deadline sẽ làm approval expired
// và trả về domain.ErrApprovalExpired, nên một "allow" muộn không bao giờ mở khóa được agent.
func (s *Service) Decide(ctx context.Context, id domain.ApprovalID, d domain.Decision, by, reason string) (domain.Approval, error) {
	now := s.clock.Now()
	var expired bool
	updated, err := s.repo.Update(ctx, id, func(a *domain.Approval) error {
		if a.Expire(now) {
			expired = true
			return nil
		}
		return a.Decide(d, by, reason, now)
	})
	if err != nil {
		return domain.Approval{}, fmt.Errorf("decide approval %s: %w", id, err)
	}
	if expired {
		s.log.WarnContext(ctx, "approval decided after deadline",
			slog.String("approval_id", string(id)),
			slog.String("decided_by", by),
		)
		return updated, ErrLateDecision
	}
	s.log.InfoContext(ctx, "approval decided",
		slog.String("approval_id", string(id)),
		slog.String("status", string(updated.Status)),
		slog.String("decided_by", by),
	)
	return updated, nil
}

// ErrLateDecision wraps domain.ErrApprovalExpired for a decision that arrived too late.
// The approval is persisted as expired before this error is returned.
//
// ErrLateDecision bọc domain.ErrApprovalExpired cho quyết định đến quá muộn.
// Approval đã được lưu ở trạng thái expired trước khi trả lỗi này.
var ErrLateDecision = fmt.Errorf("late decision: %w", domain.ErrApprovalExpired)

// ExpireDue expires every pending approval whose deadline has passed and returns how many changed.
// It keeps going after a single failure and returns all failures joined.
//
// ExpireDue cho hết hạn mọi approval pending đã quá deadline và trả về số lượng đã đổi.
// Gặp lỗi ở một bản ghi vẫn chạy tiếp và trả về tất cả lỗi gộp lại.
func (s *Service) ExpireDue(ctx context.Context) (int, error) {
	now := s.clock.Now()
	due, err := s.repo.ListPendingBefore(ctx, now)
	if err != nil {
		return 0, fmt.Errorf("list due approvals: %w", err)
	}
	var (
		changed int
		errs    []error
	)
	for _, a := range due {
		if err := ctx.Err(); err != nil {
			errs = append(errs, err)
			break
		}
		var did bool
		_, err := s.repo.Update(ctx, a.ID, func(cur *domain.Approval) error {
			did = cur.Expire(now)
			return nil
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("expire approval %s: %w", a.ID, err))
			continue
		}
		if did {
			changed++
			s.log.InfoContext(ctx, "approval expired", slog.String("approval_id", string(a.ID)))
		}
	}
	return changed, errors.Join(errs...)
}
