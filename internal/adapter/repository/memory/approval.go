// Package memory provides in-memory repositories for local runs and tests. Data is lost on restart.
// Package memory cung cấp repository in-memory cho chạy local và test. Dữ liệu mất khi restart.
package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/chjpcojit/agents-management/internal/domain"
	"github.com/chjpcojit/agents-management/internal/usecase/port"
)

var _ port.ApprovalRepository = (*ApprovalRepository)(nil)

// ApprovalRepository is a mutex-guarded map. Update holds the lock for the whole mutation,
// which gives the atomic read-modify-write the port requires.
//
// ApprovalRepository là map được bảo vệ bằng mutex. Update giữ lock suốt quá trình mutate,
// đảm bảo read-modify-write atomic như port yêu cầu.
type ApprovalRepository struct {
	mu    sync.Mutex
	items map[domain.ApprovalID]domain.Approval
}

// NewApprovalRepository returns an empty repository.
// NewApprovalRepository trả về một repository rỗng.
func NewApprovalRepository() *ApprovalRepository {
	return &ApprovalRepository{items: make(map[domain.ApprovalID]domain.Approval)}
}

// Create stores a new approval. It fails with domain.ErrAlreadyExists on a duplicate ID.
// Create lưu approval mới. Trả domain.ErrAlreadyExists nếu trùng ID.
func (r *ApprovalRepository) Create(ctx context.Context, a domain.Approval) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[a.ID]; ok {
		return fmt.Errorf("approval %s: %w", a.ID, domain.ErrAlreadyExists)
	}
	r.items[a.ID] = clone(a)
	return nil
}

// Get returns a copy of the stored approval.
// Get trả về bản copy của approval đang lưu.
func (r *ApprovalRepository) Get(ctx context.Context, id domain.ApprovalID) (domain.Approval, error) {
	if err := ctx.Err(); err != nil {
		return domain.Approval{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.items[id]
	if !ok {
		return domain.Approval{}, fmt.Errorf("approval %s: %w", id, domain.ErrNotFound)
	}
	return clone(a), nil
}

// Update runs mutate on a working copy and stores it only when mutate returns nil.
// Update chạy mutate trên bản copy và chỉ lưu khi mutate trả về nil.
func (r *ApprovalRepository) Update(ctx context.Context, id domain.ApprovalID, mutate func(*domain.Approval) error) (domain.Approval, error) {
	if err := ctx.Err(); err != nil {
		return domain.Approval{}, err
	}
	if mutate == nil {
		return domain.Approval{}, fmt.Errorf("%w: mutate is required", domain.ErrInvalidArgument)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	cur, ok := r.items[id]
	if !ok {
		return domain.Approval{}, fmt.Errorf("approval %s: %w", id, domain.ErrNotFound)
	}
	work := clone(cur)
	if err := mutate(&work); err != nil {
		return domain.Approval{}, err
	}
	if work.ID != id {
		return domain.Approval{}, fmt.Errorf("%w: mutate must not change the id", domain.ErrInvalidArgument)
	}
	r.items[id] = work
	return clone(work), nil
}

// ListPendingBefore returns pending approvals whose deadline is not after t, oldest deadline first.
// ListPendingBefore trả về các approval pending có deadline không sau t, deadline cũ nhất trước.
func (r *ApprovalRepository) ListPendingBefore(ctx context.Context, t time.Time) ([]domain.Approval, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.Lock()
	out := make([]domain.Approval, 0)
	for _, a := range r.items {
		if a.Status == domain.ApprovalPending && !a.Deadline.After(t) {
			out = append(out, clone(a))
		}
	}
	r.mu.Unlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Deadline.Before(out[j].Deadline) })
	return out, nil
}

// clone deep-copies the slice fields so callers cannot mutate stored state.
// clone deep-copy các field slice để caller không sửa được state đang lưu.
func clone(a domain.Approval) domain.Approval {
	if a.Input != nil {
		a.Input = append([]byte(nil), a.Input...)
	}
	return a
}
