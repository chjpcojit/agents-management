package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/chjpcojit/agents-management/internal/adapter/repository/memory"
	"github.com/chjpcojit/agents-management/internal/adapter/runtime/claudecli"
	"github.com/chjpcojit/agents-management/internal/adapter/runtime/claudecli/protocol"
	"github.com/chjpcojit/agents-management/internal/adapter/system"
	"github.com/chjpcojit/agents-management/internal/delivery/mcpbus"
	"github.com/chjpcojit/agents-management/internal/domain"
	"github.com/chjpcojit/agents-management/internal/usecase/approval"
)

// Policy decides an approval automatically. ok=false leaves it pending (used to test expiry).
// Policy tự động quyết định một approval. ok=false để nguyên pending (dùng để test hết hạn).
type Policy func(a domain.Approval) (d domain.Decision, reason string, ok bool)

// AllowAll approves everything. / AllowAll duyệt mọi thứ.
func AllowAll(domain.Approval) (domain.Decision, string, bool) { return domain.DecisionAllow, "", true }

// busMessage is one send_message call captured by the harness.
// busMessage là một lần gọi send_message mà harness bắt được.
type busMessage struct {
	From domain.AgentID
	To   string
	Text string
}

// harnessOptions configures one harness. / harnessOptions cấu hình một harness.
type harnessOptions struct {
	ClaudeBin       string
	Model           string
	ApprovalTimeout time.Duration
	Policy          Policy
	// OnApproval is called after the policy ran, for interactive printing.
	// OnApproval được gọi sau khi policy chạy, để in ra ở chế độ tương tác.
	OnApproval func(a domain.Approval)
	Log        *slog.Logger
}

// harness wires the real approval use case, the real MCP server on a loopback port and the runner.
// It is the composition root of the spike, so it may import every layer.
//
// harness nối use case approval thật, MCP server thật trên cổng loopback và runner.
// Đây là composition root của spike nên được import mọi layer.
type harness struct {
	opts      harnessOptions
	approvals *approval.Service
	bus       *mcpbus.Server
	listener  net.Listener
	srv       *http.Server
	baseURL   string
	log       *slog.Logger

	mu        sync.Mutex
	opened    []domain.Approval
	argKeys   map[domain.AgentID][][]string
	messages  []busMessage
	agents    []*agent
	stopSweep context.CancelFunc
	sweepDone chan struct{}
}

func newHarness(opts harnessOptions) (*harness, error) {
	if opts.ApprovalTimeout <= 0 {
		opts.ApprovalTimeout = 5 * time.Minute
	}
	if opts.Policy == nil {
		opts.Policy = AllowAll
	}
	if opts.Log == nil {
		opts.Log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	h := &harness{opts: opts, log: opts.Log, argKeys: make(map[domain.AgentID][][]string)}

	svc, err := approval.NewService(memory.NewApprovalRepository(), system.Clock{}, system.UUIDGenerator{},
		opts.ApprovalTimeout, opts.Log.With(slog.String("component", "approval")), approval.WithPollInterval(100*time.Millisecond))
	if err != nil {
		return nil, fmt.Errorf("approval service: %w", err)
	}
	h.approvals = svc

	bus, err := mcpbus.New(svc, h.sink, mcpbus.Hooks{
		OnApprovalCall:   h.onApprovalCall,
		OnApprovalOpened: h.onApprovalOpened,
	}, opts.Log.With(slog.String("component", "mcpbus")))
	if err != nil {
		return nil, fmt.Errorf("mcp server: %w", err)
	}
	h.bus = bus

	// Loopback only, random port: the MCP endpoint is never reachable from outside the machine.
	// Chỉ loopback, cổng ngẫu nhiên: endpoint MCP không bao giờ truy cập được từ ngoài máy.
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen loopback: %w", err)
	}
	h.listener = ln
	h.baseURL = "http://" + ln.Addr().String()
	mux := http.NewServeMux()
	mux.Handle(mcpbus.PathPrefix, bus.Handler())
	h.srv = &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		if err := h.srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			h.log.Error("mcp http server stopped", slog.Any("error", err))
		}
	}()

	// The sweeper is the second line of default-deny, next to AwaitDecision's own expiry.
	// Sweeper là lớp default-deny thứ hai, bên cạnh việc AwaitDecision tự cho hết hạn.
	ctx, cancel := context.WithCancel(context.Background())
	h.stopSweep = cancel
	h.sweepDone = make(chan struct{})
	go func() {
		defer close(h.sweepDone)
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if _, err := svc.ExpireDue(ctx); err != nil && ctx.Err() == nil {
					h.log.Warn("expire due approvals", slog.Any("error", err))
				}
			}
		}
	}()
	return h, nil
}

func (h *harness) sink(_ context.Context, from domain.AgentID, to, text string) error {
	h.mu.Lock()
	h.messages = append(h.messages, busMessage{From: from, To: to, Text: text})
	h.mu.Unlock()
	return nil
}

func (h *harness) onApprovalCall(agent domain.AgentID, keys []string) {
	h.mu.Lock()
	h.argKeys[agent] = append(h.argKeys[agent], keys)
	h.mu.Unlock()
}

func (h *harness) onApprovalOpened(a domain.Approval) {
	h.mu.Lock()
	h.opened = append(h.opened, a)
	h.mu.Unlock()
	if d, reason, ok := h.opts.Policy(a); ok {
		if _, err := h.approvals.Decide(context.Background(), a.ID, d, "spike-policy", reason); err != nil {
			h.log.Warn("auto decision failed", slog.String("approval_id", string(a.ID)), slog.Any("error", err))
		}
	}
	if h.opts.OnApproval != nil {
		h.opts.OnApproval(a)
	}
}

// Opened returns the approvals opened so far, optionally for one agent.
// Opened trả về các approval đã mở, có thể lọc theo một agent.
func (h *harness) Opened(agent domain.AgentID) []domain.Approval {
	h.mu.Lock()
	defer h.mu.Unlock()
	var out []domain.Approval
	for _, a := range h.opened {
		if agent == "" || a.AgentID == agent {
			out = append(out, a)
		}
	}
	return out
}

// ArgKeys returns the argument key sets seen on approval_prompt for agent (Q-2).
// ArgKeys trả về các tập key đối số thấy ở approval_prompt của agent (Q-2).
func (h *harness) ArgKeys(agent domain.AgentID) [][]string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([][]string(nil), h.argKeys[agent]...)
}

// Messages returns the captured bus messages. / Messages trả về các bus message đã bắt được.
func (h *harness) Messages() []busMessage {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]busMessage(nil), h.messages...)
}

// Final reads the stored, current state of an approval. / Final đọc trạng thái hiện tại đã lưu của approval.
func (h *harness) Final(ctx context.Context, id domain.ApprovalID) (domain.Approval, error) {
	return h.approvals.AwaitDecision(ctx, id)
}

// agentSpec describes one agent to start. / agentSpec mô tả một agent cần khởi động.
type agentSpec struct {
	ID     string
	Dir    string
	Resume string
	Raw    io.Writer // optional NDJSON capture / tùy chọn: ghi lại NDJSON
}

// StartAgent registers the agent on the MCP server and spawns its process.
// StartAgent đăng ký agent trên MCP server rồi spawn process của nó.
func (h *harness) StartAgent(spec agentSpec) (*agent, error) {
	token, err := h.bus.Register(domain.AgentID(spec.ID))
	if err != nil {
		return nil, err
	}
	cfg := claudecli.Config{
		AgentID:  spec.ID,
		Bin:      h.opts.ClaudeBin,
		Dir:      spec.Dir,
		Model:    h.opts.Model,
		MCPURL:   mcpbus.AgentURL(h.baseURL, domain.AgentID(spec.ID)),
		MCPToken: token,
	}
	if spec.Resume != "" {
		cfg.Resume = spec.Resume
	} else {
		cfg.SessionID = system.UUIDGenerator{}.NewID()
	}
	r, err := claudecli.Start(cfg, h.log.With(slog.String("component", "runner")))
	if err != nil {
		h.bus.Unregister(domain.AgentID(spec.ID))
		return nil, err
	}
	a := newAgent(spec.ID, r, spec.Raw)
	h.mu.Lock()
	h.agents = append(h.agents, a)
	h.mu.Unlock()
	return a, nil
}

// Probe reports whether the MCP endpoint still answers (an unauthenticated call must get 401).
// Probe cho biết endpoint MCP còn trả lời không (gọi không token phải nhận 401).
func (h *harness) Probe(ctx context.Context) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.baseURL+mcpbus.PathPrefix+"probe", strings.NewReader("{}"))
	if err != nil {
		return 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	_ = resp.Body.Close()
	return resp.StatusCode, nil
}

// Close stops every agent, then the MCP server and the sweeper.
// Close dừng mọi agent, rồi tới MCP server và sweeper.
func (h *harness) Close() {
	h.mu.Lock()
	agents := append([]*agent(nil), h.agents...)
	h.mu.Unlock()
	for _, a := range agents {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = a.r.Close(ctx)
		cancel()
		<-a.drained
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = h.srv.Shutdown(ctx)
	h.stopSweep()
	<-h.sweepDone
}

// agent drains a runner's events into an in-memory log that scenarios can wait on.
// agent đọc hết event của runner vào log trong bộ nhớ để scenario có thể chờ trên đó.
type agent struct {
	id string
	r  *claudecli.Runner

	mu       sync.Mutex
	msgs     []timedMessage
	states   []claudecli.Event
	exited   *claudecli.Event
	changed  chan struct{}
	drained  chan struct{}
	onEvent  func(claudecli.Event)
	rawOut   io.Writer
	rawError error
}

type timedMessage struct {
	At  time.Time
	Msg protocol.Message
}

func newAgent(id string, r *claudecli.Runner, raw io.Writer) *agent {
	a := &agent{id: id, r: r, changed: make(chan struct{}), drained: make(chan struct{}), rawOut: raw}
	go a.drain()
	return a
}

func (a *agent) drain() {
	defer close(a.drained)
	for e := range a.r.Events() {
		a.mu.Lock()
		switch e.Type {
		case claudecli.EventMessage:
			a.msgs = append(a.msgs, timedMessage{At: e.At, Msg: e.Message})
			if a.rawOut != nil && a.rawError == nil {
				_, a.rawError = a.rawOut.Write(append(append([]byte(nil), e.Message.Raw...), '\n'))
			}
		case claudecli.EventState:
			a.states = append(a.states, e)
		case claudecli.EventExited:
			ev := e
			a.exited = &ev
		}
		cb := a.onEvent
		close(a.changed)
		a.changed = make(chan struct{})
		a.mu.Unlock()
		if cb != nil {
			cb(e)
		}
	}
}

// Len returns how many messages were received so far. / Len trả về số message đã nhận.
func (a *agent) Len() int { a.mu.Lock(); defer a.mu.Unlock(); return len(a.msgs) }

// Since returns the messages from index i on. / Since trả về các message từ vị trí i trở đi.
func (a *agent) Since(i int) []timedMessage {
	a.mu.Lock()
	defer a.mu.Unlock()
	if i >= len(a.msgs) {
		return nil
	}
	return append([]timedMessage(nil), a.msgs[i:]...)
}

// Exited returns the exit event, if the process has exited. / Exited trả về event thoát nếu process đã thoát.
func (a *agent) Exited() *claudecli.Event { a.mu.Lock(); defer a.mu.Unlock(); return a.exited }

// States returns the state events so far. / States trả về các event trạng thái đã có.
func (a *agent) States() []claudecli.Event {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]claudecli.Event(nil), a.states...)
}

// WaitFor blocks until cond holds over the messages since index from, the process exits, or ctx ends.
// WaitFor chặn tới khi cond đúng trên các message từ vị trí from, process thoát, hoặc ctx kết thúc.
func (a *agent) WaitFor(ctx context.Context, from int, cond func([]timedMessage) bool) ([]timedMessage, error) {
	for {
		a.mu.Lock()
		var msgs []timedMessage
		if from < len(a.msgs) {
			msgs = append([]timedMessage(nil), a.msgs[from:]...)
		}
		exited := a.exited != nil
		ch := a.changed
		a.mu.Unlock()
		if cond(msgs) {
			return msgs, nil
		}
		if exited {
			return msgs, claudecli.ErrProcessExited
		}
		select {
		case <-ctx.Done():
			return msgs, ctx.Err()
		case <-ch:
		}
	}
}

// countResults counts result messages. / countResults đếm số message result.
func countResults(msgs []timedMessage) int {
	n := 0
	for _, m := range msgs {
		if m.Msg.Kind == protocol.KindResult {
			n++
		}
	}
	return n
}

func hasResults(n int) func([]timedMessage) bool {
	return func(m []timedMessage) bool { return countResults(m) >= n }
}

// Turn sends prompt and waits for its result. / Turn gửi prompt và chờ result của nó.
func (a *agent) Turn(ctx context.Context, prompt string, timeout time.Duration) ([]timedMessage, error) {
	from := a.Len()
	if err := a.r.Send(prompt); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return a.WaitFor(ctx, from, hasResults(1))
}

// lastResult returns the last result in msgs, or nil. / lastResult trả về result cuối trong msgs, hoặc nil.
func lastResult(msgs []timedMessage) *protocol.Result {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Msg.Kind == protocol.KindResult && msgs[i].Msg.Result != nil {
			return msgs[i].Msg.Result
		}
	}
	return nil
}

// makeSandbox creates a fresh directory under root (or the OS temp dir).
// makeSandbox tạo thư mục mới dưới root (hoặc thư mục temp của OS).
func makeSandbox(root, name string) (string, error) {
	if root != "" {
		if err := os.MkdirAll(root, 0o750); err != nil {
			return "", fmt.Errorf("create sandbox root: %w", err)
		}
	}
	dir, err := os.MkdirTemp(root, "spike-"+name+"-")
	if err != nil {
		return "", fmt.Errorf("create sandbox: %w", err)
	}
	return claudecli.SandboxDir(dir)
}
