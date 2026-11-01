// Package port declares the interfaces use cases depend on. Adapters implement them.
// Dependencies point inward: adapters import port, never the other way round.
//
// Package port khai báo các interface mà use case phụ thuộc. Adapter implement chúng.
// Chiều phụ thuộc hướng vào trong: adapter import port, không bao giờ ngược lại.
package port

import (
	"context"
	"time"

	"github.com/chjpcojit/agents-management/internal/domain"
)

// Clock returns the current time. Injected so rules on deadlines are testable.
// Clock trả về thời gian hiện tại. Được inject để test được các rule về deadline.
type Clock interface {
	Now() time.Time
}

// IDGenerator returns a new globally unique identifier.
// IDGenerator trả về một định danh duy nhất toàn cục.
type IDGenerator interface {
	NewID() string
}

// ApprovalRepository stores approvals.
// Update must apply mutate atomically against the latest stored version and persist the result
// only when mutate returns nil, so two concurrent decisions cannot both win.
//
// ApprovalRepository lưu trữ approval.
// Update phải áp dụng mutate một cách atomic trên phiên bản mới nhất đang lưu và chỉ persist khi
// mutate trả về nil, để hai quyết định đồng thời không thể cùng thắng.
type ApprovalRepository interface {
	Create(ctx context.Context, a domain.Approval) error
	Get(ctx context.Context, id domain.ApprovalID) (domain.Approval, error)
	Update(ctx context.Context, id domain.ApprovalID, mutate func(*domain.Approval) error) (domain.Approval, error)
	ListPendingBefore(ctx context.Context, t time.Time) ([]domain.Approval, error)
}
