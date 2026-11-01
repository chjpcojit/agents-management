package mcpbus_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/chjpcojit/agents-management/internal/adapter/repository/memory"
	"github.com/chjpcojit/agents-management/internal/adapter/system"
	"github.com/chjpcojit/agents-management/internal/delivery/mcpbus"
	"github.com/chjpcojit/agents-management/internal/domain"
	"github.com/chjpcojit/agents-management/internal/usecase/approval"
)

type bearer struct {
	token string
	base  http.RoundTripper
}

func (b bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.token)
	return b.base.RoundTrip(r)
}

type fixture struct {
	svc    *approval.Service
	bus    *mcpbus.Server
	srv    *httptest.Server
	opened chan domain.Approval
	keys   chan []string
	mu     sync.Mutex
	sent   []string
}

func newFixture(t *testing.T, timeout time.Duration) *fixture {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc, err := approval.NewService(memory.NewApprovalRepository(), system.Clock{}, system.UUIDGenerator{}, timeout, log, approval.WithPollInterval(10*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{svc: svc, opened: make(chan domain.Approval, 8), keys: make(chan []string, 8)}
	bus, err := mcpbus.New(svc, func(_ context.Context, from domain.AgentID, to, text string) error {
		f.mu.Lock()
		f.sent = append(f.sent, string(from)+"->"+to+":"+text)
		f.mu.Unlock()
		return nil
	}, mcpbus.Hooks{
		OnApprovalOpened: func(a domain.Approval) { f.opened <- a },
		OnApprovalCall:   func(_ domain.AgentID, k []string) { f.keys <- k },
	}, log)
	if err != nil {
		t.Fatal(err)
	}
	f.bus = bus
	mux := http.NewServeMux()
	mux.Handle(mcpbus.PathPrefix, bus.Handler())
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fixture) connect(t *testing.T, agent domain.AgentID, token string) (*mcp.ClientSession, error) {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:   mcpbus.AgentURL(f.srv.URL, agent),
		HTTPClient: &http.Client{Transport: bearer{token: token, base: http.DefaultTransport}},
	}, nil)
}

func callApproval(t *testing.T, cs *mcp.ClientSession) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      mcpbus.ToolApprovalPrompt,
		Arguments: map[string]any{"tool_name": "Bash", "input": map[string]any{"command": "touch x"}, "tool_use_id": "toolu_1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestApprovalPromptAllow(t *testing.T) {
	f := newFixture(t, time.Minute)
	tok, _ := f.bus.Register("agent-a")
	cs, err := f.connect(t, "agent-a", tok)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cs.Close() }()

	go func() {
		a := <-f.opened
		if a.AgentID != "agent-a" || a.ToolName != "Bash" || a.Risk != domain.RiskHigh {
			t.Errorf("unexpected approval %+v", a)
		}
		_, _ = f.svc.Decide(context.Background(), a.ID, domain.DecisionAllow, "duc", "")
	}()
	out := callApproval(t, cs)
	if out["behavior"] != "allow" {
		t.Fatalf("want allow, got %v", out)
	}
	if in, _ := out["updatedInput"].(map[string]any); in["command"] != "touch x" {
		t.Fatalf("allow must echo the original input, got %v", out["updatedInput"])
	}
	if keys := <-f.keys; strings.Join(keys, ",") != "input,tool_name,tool_use_id" {
		t.Fatalf("hook must report argument keys, got %v", keys)
	}
}

func TestApprovalPromptDenyCarriesReason(t *testing.T) {
	f := newFixture(t, time.Minute)
	tok, _ := f.bus.Register("agent-a")
	cs, _ := f.connect(t, "agent-a", tok)
	defer func() { _ = cs.Close() }()
	go func() {
		a := <-f.opened
		_, _ = f.svc.Decide(context.Background(), a.ID, domain.DecisionDeny, "duc", "not in this repo")
	}()
	out := callApproval(t, cs)
	if out["behavior"] != "deny" || !strings.Contains(out["message"].(string), "not in this repo") {
		t.Fatalf("got %v", out)
	}
}

func TestApprovalPromptExpiresToDeny(t *testing.T) {
	f := newFixture(t, 50*time.Millisecond)
	tok, _ := f.bus.Register("agent-a")
	cs, _ := f.connect(t, "agent-a", tok)
	defer func() { _ = cs.Close() }()
	out := callApproval(t, cs)
	if out["behavior"] != "deny" || !strings.Contains(out["message"].(string), "deadline") {
		t.Fatalf("got %v", out)
	}
}

func TestRejectsWrongTokenAndUnknownAgent(t *testing.T) {
	f := newFixture(t, time.Minute)
	tokA, _ := f.bus.Register("agent-a")
	_, _ = f.bus.Register("agent-b")
	if _, err := f.connect(t, "agent-b", tokA); err == nil {
		t.Fatal("agent-a's token must not open agent-b's endpoint")
	}
	if _, err := f.connect(t, "ghost", tokA); err == nil {
		t.Fatal("unknown agent must be rejected")
	}
	f.bus.Unregister("agent-a")
	if _, err := f.connect(t, "agent-a", tokA); err == nil {
		t.Fatal("unregistered agent's token must stop working")
	}
}

func TestTwoAgentsAreRoutedSeparately(t *testing.T) {
	f := newFixture(t, time.Minute)
	tokA, _ := f.bus.Register("agent-a")
	tokB, _ := f.bus.Register("agent-b")
	csA, _ := f.connect(t, "agent-a", tokA)
	csB, _ := f.connect(t, "agent-b", tokB)
	defer func() { _ = csA.Close() }()
	defer func() { _ = csB.Close() }()

	go func() {
		for i := 0; i < 2; i++ {
			a := <-f.opened
			d := domain.DecisionDeny
			if a.AgentID == "agent-a" {
				d = domain.DecisionAllow
			}
			_, _ = f.svc.Decide(context.Background(), a.ID, d, "duc", "")
		}
	}()
	var wg sync.WaitGroup
	results := map[string]any{}
	var mu sync.Mutex
	for name, cs := range map[string]*mcp.ClientSession{"agent-a": csA, "agent-b": csB} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out := callApproval(t, cs)
			mu.Lock()
			results[name] = out["behavior"]
			mu.Unlock()
		}()
	}
	wg.Wait()
	if results["agent-a"] != "allow" || results["agent-b"] != "deny" {
		t.Fatalf("decisions routed to the wrong agent: %v", results)
	}
}

func TestSendMessageStub(t *testing.T) {
	f := newFixture(t, time.Minute)
	tok, _ := f.bus.Register("agent-a")
	cs, _ := f.connect(t, "agent-a", tok)
	defer func() { _ = cs.Close() }()
	ctx := context.Background()
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: mcpbus.ToolSendMessage, Arguments: map[string]any{"to": "agent-b", "text": "hello"}})
	if err != nil || res.IsError {
		t.Fatalf("send failed: %v %+v", err, res)
	}
	res, _ = cs.CallTool(ctx, &mcp.CallToolParams{Name: mcpbus.ToolSendMessage, Arguments: map[string]any{"to": "agent-b", "text": ""}})
	if !res.IsError {
		t.Fatal("empty text must be rejected")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.sent) != 1 || f.sent[0] != "agent-a->agent-b:hello" {
		t.Fatalf("got %v", f.sent)
	}
}
