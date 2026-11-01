package approval_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chjpcojit/agents-management/internal/adapter/repository/memory"
	"github.com/chjpcojit/agents-management/internal/domain"
	"github.com/chjpcojit/agents-management/internal/usecase/approval"
)

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time          { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *fakeClock) Advance(d time.Duration) { c.mu.Lock(); c.now = c.now.Add(d); c.mu.Unlock() }

type seqIDs struct{ n atomic.Int64 }

func (s *seqIDs) NewID() string { return fmt.Sprintf("ap-%d", s.n.Add(1)) }

func newService(t *testing.T) (*approval.Service, *fakeClock) {
	t.Helper()
	clk := &fakeClock{now: time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)}
	svc, err := approval.NewService(memory.NewApprovalRepository(), clk, &seqIDs{}, time.Minute, nil)
	if err != nil {
		t.Fatal(err)
	}
	return svc, clk
}

func toolRequest() approval.RequestInput {
	return approval.RequestInput{
		AgentID: "agent-a", Kind: domain.ApprovalKindTool, ToolName: "Bash",
		Input: json.RawMessage(`{"command":"touch x"}`), Risk: domain.RiskHigh,
	}
}

func TestNewService_RejectsMissingDeps(t *testing.T) {
	if _, err := approval.NewService(nil, &fakeClock{}, &seqIDs{}, time.Minute, nil); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("want ErrInvalidArgument, got %v", err)
	}
}

func TestRequestThenAllow(t *testing.T) {
	svc, clk := newService(t)
	ctx := context.Background()
	a, err := svc.Request(ctx, toolRequest())
	if err != nil {
		t.Fatal(err)
	}
	clk.Advance(10 * time.Second)
	got, err := svc.Decide(ctx, a.ID, domain.DecisionAllow, "duc", "")
	if err != nil || !got.IsAllowed() {
		t.Fatalf("want allowed, got %+v err %v", got, err)
	}
}

func TestLateDecisionIsDenied(t *testing.T) {
	svc, clk := newService(t)
	ctx := context.Background()
	a, _ := svc.Request(ctx, toolRequest())
	clk.Advance(2 * time.Minute)
	got, err := svc.Decide(ctx, a.ID, domain.DecisionAllow, "duc", "")
	if !errors.Is(err, approval.ErrLateDecision) || !errors.Is(err, domain.ErrApprovalExpired) {
		t.Fatalf("want ErrLateDecision, got %v", err)
	}
	if got.Status != domain.ApprovalExpired || got.IsAllowed() {
		t.Fatalf("late allow must persist as expired, got %s", got.Status)
	}
}

func TestDecideUnknownID(t *testing.T) {
	svc, _ := newService(t)
	if _, err := svc.Decide(context.Background(), "nope", domain.DecisionAllow, "duc", ""); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestConcurrentDecisionsOnlyOneWins(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()
	a, _ := svc.Request(ctx, toolRequest())

	var wins atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			d := domain.DecisionAllow
			if i%2 == 0 {
				d = domain.DecisionDeny
			}
			if _, err := svc.Decide(ctx, a.ID, d, "duc", ""); err == nil {
				wins.Add(1)
			} else if !errors.Is(err, domain.ErrApprovalAlreadyDecided) {
				t.Errorf("unexpected error: %v", err)
			}
		}(i)
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("want exactly 1 winning decision, got %d", wins.Load())
	}
}

func TestExpireDue(t *testing.T) {
	svc, clk := newService(t)
	ctx := context.Background()
	first, _ := svc.Request(ctx, toolRequest())
	clk.Advance(30 * time.Second)
	second, _ := svc.Request(ctx, toolRequest())
	clk.Advance(31 * time.Second) // first is due, second is not / first quá hạn, second chưa

	n, err := svc.ExpireDue(ctx)
	if err != nil || n != 1 {
		t.Fatalf("want 1 expired, got %d err %v", n, err)
	}
	if _, err := svc.Decide(ctx, first.ID, domain.DecisionAllow, "duc", ""); !errors.Is(err, domain.ErrApprovalAlreadyDecided) {
		t.Fatalf("expired approval must stay closed, got %v", err)
	}
	if got, err := svc.Decide(ctx, second.ID, domain.DecisionAllow, "duc", ""); err != nil || !got.IsAllowed() {
		t.Fatalf("second should still be decidable, got %v", err)
	}
}

func TestAwaitDecisionReturnsDecision(t *testing.T) {
	clk := &fakeClock{now: time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)}
	svc, _ := approval.NewService(memory.NewApprovalRepository(), clk, &seqIDs{}, time.Minute, nil, approval.WithPollInterval(5*time.Millisecond))
	ctx := context.Background()
	a, _ := svc.Request(ctx, toolRequest())
	go func() {
		time.Sleep(20 * time.Millisecond)
		_, _ = svc.Decide(ctx, a.ID, domain.DecisionDeny, "duc", "no")
	}()
	got, err := svc.AwaitDecision(ctx, a.ID)
	if err != nil || got.Status != domain.ApprovalDenied || got.Reason != "no" {
		t.Fatalf("got %+v err %v", got, err)
	}
}

func TestAwaitDecisionExpiresByItself(t *testing.T) {
	clk := &fakeClock{now: time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)}
	svc, _ := approval.NewService(memory.NewApprovalRepository(), clk, &seqIDs{}, time.Minute, nil, approval.WithPollInterval(5*time.Millisecond))
	ctx := context.Background()
	a, _ := svc.Request(ctx, toolRequest())
	go func() { time.Sleep(20 * time.Millisecond); clk.Advance(time.Minute) }()
	got, err := svc.AwaitDecision(ctx, a.ID)
	if err != nil || got.Status != domain.ApprovalExpired || got.IsAllowed() {
		t.Fatalf("got %+v err %v", got, err)
	}
}

func TestAwaitDecisionHonoursContext(t *testing.T) {
	svc, _ := newService(t)
	a, _ := svc.Request(context.Background(), toolRequest())
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if _, err := svc.AwaitDecision(ctx, a.ID); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want deadline exceeded, got %v", err)
	}
}

func TestPending(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()
	a, _ := svc.Request(ctx, toolRequest())
	b, _ := svc.Request(ctx, toolRequest())
	_, _ = svc.Decide(ctx, b.ID, domain.DecisionAllow, "duc", "")
	got, err := svc.Pending(ctx)
	if err != nil || len(got) != 1 || got[0].ID != a.ID {
		t.Fatalf("want only %s pending, got %+v err %v", a.ID, got, err)
	}
}
