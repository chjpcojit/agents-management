package domain_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/chjpcojit/agents-management/internal/domain"
)

var t0 = time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)

func validParams() domain.NewApprovalParams {
	return domain.NewApprovalParams{
		ID: "ap-1", AgentID: "agent-a", Kind: domain.ApprovalKindTool, ToolName: "Bash",
		Input: json.RawMessage(`{"command":"ls"}`), Risk: domain.RiskLow, Now: t0, Timeout: time.Minute,
	}
}

func TestNewApproval_Valid(t *testing.T) {
	a, err := domain.NewApproval(validParams())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Status != domain.ApprovalPending || !a.Deadline.Equal(t0.Add(time.Minute)) {
		t.Fatalf("got status %s deadline %s", a.Status, a.Deadline)
	}
}

func TestNewApproval_Invalid(t *testing.T) {
	cases := map[string]func(*domain.NewApprovalParams){
		"missing id":        func(p *domain.NewApprovalParams) { p.ID = " " },
		"missing agent":     func(p *domain.NewApprovalParams) { p.AgentID = "" },
		"tool without name": func(p *domain.NewApprovalParams) { p.ToolName = "" },
		"unknown kind":      func(p *domain.NewApprovalParams) { p.Kind = "x" },
		"unknown risk":      func(p *domain.NewApprovalParams) { p.Risk = "" },
		"zero timeout":      func(p *domain.NewApprovalParams) { p.Timeout = 0 },
		"zero now":          func(p *domain.NewApprovalParams) { p.Now = time.Time{} },
		"bad json":          func(p *domain.NewApprovalParams) { p.Input = json.RawMessage(`{`) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			p := validParams()
			mutate(&p)
			if _, err := domain.NewApproval(p); !errors.Is(err, domain.ErrInvalidArgument) {
				t.Fatalf("want ErrInvalidArgument, got %v", err)
			}
		})
	}
}

func TestNewApproval_CopiesInput(t *testing.T) {
	p := validParams()
	a, _ := domain.NewApproval(p)
	p.Input[2] = 'X'
	if string(a.Input) != `{"command":"ls"}` {
		t.Fatalf("input aliased caller slice: %s", a.Input)
	}
}

func TestDecide(t *testing.T) {
	tests := []struct {
		name    string
		d       domain.Decision
		by      string
		at      time.Time
		want    domain.ApprovalStatus
		wantErr error
	}{
		{"allow before deadline", domain.DecisionAllow, "duc", t0.Add(30 * time.Second), domain.ApprovalAllowed, nil},
		{"deny before deadline", domain.DecisionDeny, "duc", t0.Add(30 * time.Second), domain.ApprovalDenied, nil},
		{"at deadline is late", domain.DecisionAllow, "duc", t0.Add(time.Minute), domain.ApprovalPending, domain.ErrApprovalExpired},
		{"missing decider", domain.DecisionAllow, "", t0, domain.ApprovalPending, domain.ErrInvalidArgument},
		{"unknown decision", "maybe", "duc", t0, domain.ApprovalPending, domain.ErrInvalidArgument},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a, _ := domain.NewApproval(validParams())
			err := a.Decide(tc.d, tc.by, "", tc.at)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("want err %v, got %v", tc.wantErr, err)
			}
			if a.Status != tc.want {
				t.Fatalf("want status %s, got %s", tc.want, a.Status)
			}
		})
	}
}

func TestDecide_Twice(t *testing.T) {
	a, _ := domain.NewApproval(validParams())
	if err := a.Decide(domain.DecisionDeny, "duc", "no", t0); err != nil {
		t.Fatal(err)
	}
	if err := a.Decide(domain.DecisionAllow, "duc", "", t0); !errors.Is(err, domain.ErrApprovalAlreadyDecided) {
		t.Fatalf("want ErrApprovalAlreadyDecided, got %v", err)
	}
	if a.IsAllowed() {
		t.Fatal("second decision must not flip a deny to allow")
	}
}

func TestExpire(t *testing.T) {
	a, _ := domain.NewApproval(validParams())
	if a.Expire(t0.Add(59 * time.Second)) {
		t.Fatal("expired before deadline")
	}
	if !a.Expire(t0.Add(time.Minute)) || a.Status != domain.ApprovalExpired || a.IsAllowed() || !a.IsTerminal() {
		t.Fatalf("want expired and denied, got %+v", a)
	}
	if a.Expire(t0.Add(2 * time.Minute)) {
		t.Fatal("expire must be idempotent")
	}
}

func TestClassifyRisk(t *testing.T) {
	for tool, want := range map[string]domain.RiskLevel{
		"Read": domain.RiskLow, "Grep": domain.RiskLow,
		"Bash": domain.RiskHigh, "Write": domain.RiskHigh, "Edit": domain.RiskHigh, "WebFetch": domain.RiskHigh,
		"mcp__x__y": domain.RiskHigh, "": domain.RiskHigh,
	} {
		if got := domain.ClassifyRisk(tool); got != want {
			t.Errorf("%q: want %s, got %s", tool, want, got)
		}
	}
}
