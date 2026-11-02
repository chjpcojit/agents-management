// Package claudecli drives one `claude -p` process per agent over stream-json stdin/stdout.
// It only uses documented CLI surface: flags, stream-json, MCP permission prompt tool and settings.
//
// Package claudecli điều khiển mỗi agent bằng một process `claude -p` qua stdin/stdout stream-json.
// Chỉ dùng phần CLI có docs: flags, stream-json, MCP permission prompt tool và settings.
package claudecli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/chjpcojit/agents-management/internal/adapter/runtime/claudecli/protocol"
)

// State is the lifecycle state of one agent process.
// State là trạng thái vòng đời của một process agent.
type State string

const (
	StateIdle    State = "idle"
	StateRunning State = "running"
	StateStopped State = "stopped"
	StateFailed  State = "failed"
)

// Errors returned by Runner methods.
// Các lỗi do method của Runner trả về.
var (
	ErrProcessExited = errors.New("agent process has exited")
	ErrInvalidConfig = errors.New("invalid runner config")
)

// MCPServerName is the name the orchestrator's MCP server gets inside the agent session.
// Tool names seen by Claude are therefore mcp__orchestrator__<tool>.
//
// MCPServerName là tên MCP server của orchestrator bên trong session của agent.
// Vì vậy tên tool Claude nhìn thấy là mcp__orchestrator__<tool>.
const MCPServerName = "orchestrator"

// PermissionPromptTool is the tool Claude Code calls for every permission decision.
// PermissionPromptTool là tool Claude Code gọi cho mọi quyết định cấp quyền.
const PermissionPromptTool = "mcp__" + MCPServerName + "__approval_prompt"

// AskTools are the built-in tools that change the machine, reach the network, spawn work or schedule
// it. Each is forced through the permission prompt tool. Read-only tools are left to Claude Code.
// A new mutating tool in a later claude version is not covered until added here: spike checks every
// tool_result's permission source to catch it (see cmd/spike).
//
// AskTools là các tool built-in làm thay đổi máy, ra mạng, sinh việc hoặc lên lịch việc. Mỗi tool bị
// buộc đi qua permission prompt tool. Tool chỉ đọc để Claude Code tự xử lý. Tool ghi mới ở version
// claude sau chưa được phủ cho tới khi thêm vào đây: spike kiểm nguồn quyết định của mọi tool_result
// để bắt trường hợp đó (xem cmd/spike).
var AskTools = []string{
	"Bash", "Edit", "Write", "NotebookEdit", "WebFetch", "WebSearch",
	"Task", "Skill", "Workflow", "Monitor", "TaskStop",
	"CronCreate", "CronDelete", "ScheduleWakeup", "RemoteTrigger", "PushNotification",
	"EnterWorktree", "ExitWorktree", "DesignSync",
}

// maxLineBytes caps one stdout line; longer lines are truncated and reported as unknown.
// maxLineBytes giới hạn một dòng stdout; dòng dài hơn bị cắt và báo là unknown.
const maxLineBytes = 16 << 20

// Config describes one agent process.
// Config mô tả một process agent.
type Config struct {
	AgentID  string
	Bin      string // path or name of the claude binary / đường dẫn hoặc tên binary claude
	Dir      string // working directory, must exist / thư mục làm việc, phải tồn tại
	Model    string // optional / tùy chọn
	MCPURL   string // orchestrator MCP endpoint for this agent / endpoint MCP của orchestrator cho agent này
	MCPToken string // bearer token proving the agent's identity / bearer token chứng minh danh tính agent

	// Exactly one of SessionID (new session) or Resume (existing session) must be set.
	// Phải set đúng một trong SessionID (session mới) hoặc Resume (session cũ).
	SessionID string
	Resume    string

	// ToolTimeout is the longest Claude Code waits on one call to the orchestrator MCP server, so it
	// must exceed the approval deadline. Zero keeps Claude Code's default (60s seen in spike T7).
	// ToolTimeout là thời gian tối đa Claude Code chờ một lệnh gọi tới MCP server của orchestrator, nên
	// phải dài hơn deadline duyệt. Bằng 0 thì giữ mặc định của Claude Code (60s, thấy ở spike T7).
	ToolTimeout time.Duration

	// FirstEventTimeout bounds the wait for any output after a prompt is sent. Default 90s.
	// FirstEventTimeout giới hạn thời gian chờ output đầu tiên sau khi gửi prompt. Mặc định 90s.
	FirstEventTimeout time.Duration

	// ExtraEnv adds KEY=VALUE entries on top of the allowlist (tests, explicit opt-ins).
	// ExtraEnv thêm các cặp KEY=VALUE ngoài allowlist (cho test hoặc khi chủ động bật).
	ExtraEnv []string
	// ExtraArgs are appended after the fixed flags. / ExtraArgs được nối sau các flag cố định.
	ExtraArgs []string
}

func (c *Config) validate() error {
	var problems []string
	if strings.TrimSpace(c.AgentID) == "" {
		problems = append(problems, "agent id is required")
	}
	if c.Bin == "" {
		c.Bin = "claude"
	}
	if c.Dir == "" {
		problems = append(problems, "dir is required")
	} else if fi, err := os.Stat(c.Dir); err != nil || !fi.IsDir() {
		problems = append(problems, fmt.Sprintf("dir %q is not a directory", c.Dir))
	}
	if (c.SessionID == "") == (c.Resume == "") {
		problems = append(problems, "set exactly one of session id or resume")
	}
	if c.MCPURL == "" || c.MCPToken == "" {
		problems = append(problems, "mcp url and token are required")
	}
	if c.FirstEventTimeout == 0 {
		c.FirstEventTimeout = 90 * time.Second
	}
	if len(problems) > 0 {
		return fmt.Errorf("%w: %s", ErrInvalidConfig, strings.Join(problems, "; "))
	}
	return nil
}

// EventType classifies a Runner event.
// EventType phân loại event của Runner.
type EventType string

const (
	EventMessage EventType = "message" // one decoded stdout line / một dòng stdout đã decode
	EventState   EventType = "state"   // state changed / trạng thái thay đổi
	EventExited  EventType = "exited"  // process exited; last event / process đã thoát; event cuối cùng
)

// Event is what a Runner emits, in order.
// Event là những gì Runner phát ra, theo đúng thứ tự.
type Event struct {
	AgentID  string
	At       time.Time
	Type     EventType
	Message  protocol.Message
	State    State
	Reason   string
	ExitCode int
}

// Runner owns exactly one `claude` process. Events must be drained by the caller:
// the reader blocks when the channel is full, which back-pressures the process instead of dropping output.
//
// Runner sở hữu đúng một process `claude`. Caller phải đọc hết Events:
// reader sẽ chặn khi channel đầy, tạo back-pressure lên process thay vì làm rơi output.
type Runner struct {
	cfg Config
	log *slog.Logger
	cmd *exec.Cmd

	writeMu sync.Mutex
	stdin   io.WriteCloser

	mu        sync.Mutex
	state     State
	sessionID string
	turnSeq   uint64 // increments on every Send / tăng mỗi lần Send
	sawOutput uint64 // last turnSeq that produced output / turnSeq gần nhất đã có output
	pending   int    // prompts written without a result yet / số prompt đã ghi mà chưa có result

	emitMu sync.Mutex // serialises emit against close / tuần tự hóa emit với việc đóng channel
	closed bool
	events chan Event
	done   chan struct{}
	exit   int
}

// Start spawns the process. The returned Runner is idle and ready for Send.
// Start spawn process. Runner trả về ở trạng thái idle, sẵn sàng nhận Send.
func Start(cfg Config, log *slog.Logger) (*Runner, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	if log == nil {
		log = slog.Default()
	}
	args, err := BuildArgs(cfg)
	if err != nil {
		return nil, err
	}

	// The process outlives any request; Close owns its lifetime, so no context here.
	// Process sống lâu hơn mọi request; Close quản lý vòng đời của nó nên không dùng context ở đây.
	cmd := exec.Command(cfg.Bin, args...) //nolint:noctx,gosec // binary and args come from validated config / binary và args lấy từ config đã validate
	cmd.Dir = cfg.Dir
	setProcessGroup(cmd)
	cmd.Env = AllowedEnv(os.Environ(), cfg.ExtraEnv)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", cfg.Bin, err)
	}

	sid := cfg.SessionID
	if sid == "" {
		sid = cfg.Resume
	}
	r := &Runner{
		cfg:       cfg,
		log:       log.With(slog.String("agent_id", cfg.AgentID)),
		cmd:       cmd,
		stdin:     stdin,
		state:     StateIdle,
		sessionID: sid,
		events:    make(chan Event, 1024),
		done:      make(chan struct{}),
	}
	r.log.Info("agent process started", slog.Int("pid", cmd.Process.Pid), slog.String("session_id", sid))

	var readers sync.WaitGroup
	readers.Add(2)
	go func() { defer readers.Done(); r.readStdout(stdout) }()
	go func() { defer readers.Done(); r.readStderr(stderr) }()
	go func() {
		// os/exec requires all pipe reads to finish before Wait. / os/exec yêu cầu đọc xong pipe trước khi Wait.
		readers.Wait()
		r.finish(cmd.Wait())
	}()
	return r, nil
}

// BuildArgs returns the fixed flag set for an agent. Exported for tests and the preflight report.
// BuildArgs trả về bộ flag cố định cho một agent. Export để test và để in trong preflight.
func BuildArgs(cfg Config) ([]string, error) {
	server := map[string]any{
		"type":    "http",
		"url":     cfg.MCPURL,
		"headers": map[string]string{"Authorization": "Bearer " + cfg.MCPToken},
	}
	if cfg.ToolTimeout > 0 {
		// Per-server tool-call timeout in ms. Without it Claude Code drops a waiting approval after 60s (spike T7).
		// ⚠️ Field is in the CLI's config schema but not in the public docs (A-7).
		// Timeout gọi tool theo server, đơn vị ms. Không có nó Claude Code bỏ approval đang chờ sau 60s (spike T7).
		// ⚠️ Field có trong schema config của CLI nhưng chưa có trong docs công khai (A-7).
		server["timeout"] = cfg.ToolTimeout.Milliseconds()
	}
	mcpConfig, err := json.Marshal(map[string]any{
		"mcpServers": map[string]any{MCPServerName: server},
	})
	if err != nil {
		return nil, fmt.Errorf("encode mcp config: %w", err)
	}
	settings, err := json.Marshal(map[string]any{
		// Agents talk through the orchestrator bus only. / Agent chỉ nói chuyện qua bus của orchestrator.
		"crossSessionInbound": "refuse",
		"permissions": map[string]any{
			// An agent must not rewrite its own hooks or settings: -p runs them without a trust prompt.
			// Agent không được tự sửa hooks hay settings của mình: -p chạy chúng mà không hỏi trust.
			"deny": []string{"Edit(.claude/**)", "SendMessage", "ListAgents"},
			// Ask beats allow, so these always reach the approval inbox even when the owner's own
			// ~/.claude/settings.json allows them (spike T5 saw Bash approved by a user rule).
			// Ask thắng allow, nên các tool này luôn đi qua inbox duyệt kể cả khi settings riêng của
			// owner trong ~/.claude/settings.json cho phép (spike T5 thấy Bash được rule của user duyệt thẳng).
			"ask": AskTools,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("encode settings: %w", err)
	}

	args := []string{
		"-p",
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		"--verbose",
		"--include-partial-messages",
		"--permission-mode", "default",
		"--mcp-config", string(mcpConfig),
		"--strict-mcp-config",
		"--permission-prompt-tool", PermissionPromptTool,
		"--settings", string(settings),
	}
	if cfg.Resume != "" {
		args = append(args, "--resume", cfg.Resume)
	} else {
		args = append(args, "--session-id", cfg.SessionID)
	}
	if cfg.Model != "" {
		args = append(args, "--model", cfg.Model)
	}
	return append(args, cfg.ExtraArgs...), nil
}

// allowedEnvKeys are the only variables a child inherits. ANTHROPIC_* is never passed, so the
// binary falls back to the owner's subscription login instead of an API key found in the parent env.
//
// allowedEnvKeys là các biến duy nhất process con được kế thừa. ANTHROPIC_* không bao giờ được truyền,
// để binary dùng login subscription của owner thay vì API key nằm trong env của process cha.
var allowedEnvKeys = map[string]bool{
	"HOME": true, "PATH": true, "USER": true, "LOGNAME": true, "SHELL": true, "TERM": true,
	"LANG": true, "LC_ALL": true, "LC_CTYPE": true, "TMPDIR": true, "TZ": true,
	"XDG_CONFIG_HOME": true, "XDG_RUNTIME_DIR": true, "CLAUDE_CONFIG_DIR": true,
	"HTTP_PROXY": true, "HTTPS_PROXY": true, "NO_PROXY": true,
	"http_proxy": true, "https_proxy": true, "no_proxy": true,
	"SYSTEMROOT": true, "USERPROFILE": true, "APPDATA": true, "LOCALAPPDATA": true, "TEMP": true, "TMP": true,
}

// AllowedEnv filters environ to the allowlist and appends extra.
// AllowedEnv lọc environ theo allowlist rồi nối thêm extra.
func AllowedEnv(environ, extra []string) []string {
	out := make([]string, 0, len(allowedEnvKeys)+len(extra))
	for _, kv := range environ {
		k, _, ok := strings.Cut(kv, "=")
		if ok && allowedEnvKeys[k] {
			out = append(out, kv)
		}
	}
	return append(out, extra...)
}

// Events returns the ordered event stream. It is closed after the EventExited event.
// Events trả về stream event theo thứ tự. Channel đóng sau event EventExited.
func (r *Runner) Events() <-chan Event { return r.events }

// Done is closed once the process has exited and all events were emitted.
// Done đóng khi process đã thoát và mọi event đã được phát.
func (r *Runner) Done() <-chan struct{} { return r.done }

// State returns the current state. / State trả về trạng thái hiện tại.
func (r *Runner) State() State { r.mu.Lock(); defer r.mu.Unlock(); return r.state }

// SessionID returns the session id to pass to Resume later.
// SessionID trả về session id để dùng cho Resume sau này.
func (r *Runner) SessionID() string { r.mu.Lock(); defer r.mu.Unlock(); return r.sessionID }

// Pending returns how many written prompts still wait for a result.
// Pending trả về số prompt đã ghi còn đang chờ result.
func (r *Runner) Pending() int { r.mu.Lock(); defer r.mu.Unlock(); return r.pending }

// PID returns the process id. / PID trả về process id.
func (r *Runner) PID() int { return r.cmd.Process.Pid }

// ExitCode is valid after Done is closed. / ExitCode chỉ có nghĩa sau khi Done đóng.
func (r *Runner) ExitCode() int { <-r.done; return r.exit }

// Send writes one prompt. Claude Code queues a prompt written during a running turn and answers it
// with its own result afterwards (spike T3, Q-1), so Send never blocks; the agent stays running until
// every written prompt has a result. It fails with ErrProcessExited after exit.
//
// Send ghi một prompt. Claude Code tự xếp hàng prompt ghi vào khi turn đang chạy và trả result riêng
// cho nó sau đó (spike T3, Q-1), nên Send không chặn; agent ở running tới khi mọi prompt đã ghi có
// result. Lỗi ErrProcessExited khi process đã thoát.
func (r *Runner) Send(text string) error {
	line, err := protocol.EncodeUserMessage(text)
	if err != nil {
		return err
	}
	r.mu.Lock()
	switch r.state {
	case StateStopped, StateFailed:
		r.mu.Unlock()
		return ErrProcessExited
	}
	r.pending++
	r.turnSeq++
	seq := r.turnSeq
	changed := r.setStateLocked(StateRunning)
	r.mu.Unlock()
	if changed {
		r.emit(Event{Type: EventState, State: StateRunning})
	}

	r.writeMu.Lock()
	_, err = r.stdin.Write(line)
	r.writeMu.Unlock()
	if err != nil {
		return fmt.Errorf("%w: write prompt: %w", ErrProcessExited, err)
	}
	go r.watchFirstOutput(seq)
	return nil
}

// watchFirstOutput fails the agent when a prompt produces no output at all within the timeout.
// watchFirstOutput đánh dấu agent failed khi prompt không sinh ra output nào trong thời hạn.
func (r *Runner) watchFirstOutput(seq uint64) {
	t := time.NewTimer(r.cfg.FirstEventTimeout)
	defer t.Stop()
	select {
	case <-r.done:
		return
	case <-t.C:
	}
	r.mu.Lock()
	silent := r.sawOutput < seq && r.state == StateRunning
	r.mu.Unlock()
	if silent {
		r.fail(fmt.Sprintf("no output %s after prompt", r.cfg.FirstEventTimeout))
	}
}

// Stop cancels whatever the agent is doing: SIGTERM to its process group, then SIGKILL after 3s or
// when ctx ends. There is no documented way to cancel one turn and keep the process: in spike T12,
// SIGINT during a running tool made claude exit. Callers continue the session with Resume.
//
// Stop hủy việc agent đang làm: SIGTERM cho process group, rồi SIGKILL sau 3s hoặc khi ctx kết thúc.
// Không có cách có docs để hủy một turn mà giữ process: ở spike T12, SIGINT khi tool đang chạy làm
// claude thoát. Caller tiếp tục session bằng Resume.
func (r *Runner) Stop(ctx context.Context) error {
	select {
	case <-r.done:
		return nil
	default:
	}
	r.log.Info("agent stop requested")
	_ = r.terminate()
	return r.waitOrKill(ctx)
}

// waitOrKill waits up to 3s (or ctx) for exit, then kills the group and waits.
// waitOrKill chờ tối đa 3s (hoặc ctx) cho process thoát, rồi kill cả group và chờ.
func (r *Runner) waitOrKill(ctx context.Context) error {
	t := time.NewTimer(3 * time.Second)
	defer t.Stop()
	select {
	case <-r.done:
		return nil
	case <-ctx.Done():
	case <-t.C:
	}
	if err := r.killGroup(); err != nil {
		return fmt.Errorf("kill: %w", err)
	}
	<-r.done
	return nil
}

// Close ends the process gracefully: close stdin, wait until ctx is done, then SIGTERM, then kill.
// Close kết thúc process êm: đóng stdin, chờ tới khi ctx hết, rồi SIGTERM, rồi kill.
func (r *Runner) Close(ctx context.Context) error {
	r.writeMu.Lock()
	_ = r.stdin.Close()
	r.writeMu.Unlock()

	select {
	case <-r.done:
		return nil
	case <-ctx.Done():
	}
	r.log.Warn("agent did not exit after stdin closed, terminating")
	_ = r.terminate()
	return r.waitOrKill(context.Background())
}

func (r *Runner) readStdout(stdout io.Reader) {
	br := bufio.NewReaderSize(stdout, 64<<10)
	for {
		line, truncated, err := readLine(br, maxLineBytes)
		if len(bytes.TrimSpace(line)) > 0 {
			msg := protocol.Decode(line)
			if truncated {
				msg = protocol.Message{Kind: protocol.KindUnknown, Raw: nil, DecodeError: errors.New("line exceeds limit, truncated")}
			}
			r.onMessage(msg)
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				r.log.Warn("stdout read failed", slog.Any("error", err))
			}
			return
		}
	}
}

// readLine reads one newline-terminated line of at most limit bytes; the excess is discarded.
// readLine đọc một dòng kết thúc bằng newline, tối đa limit byte; phần dư bị bỏ.
func readLine(br *bufio.Reader, limit int) (line []byte, truncated bool, err error) {
	var buf []byte
	for {
		chunk, isPrefix, err := br.ReadLine()
		if len(buf)+len(chunk) > limit {
			truncated = true
		} else {
			buf = append(buf, chunk...)
		}
		if err != nil {
			return buf, truncated, err
		}
		if !isPrefix {
			return buf, truncated, nil
		}
	}
}

func (r *Runner) onMessage(msg protocol.Message) {
	r.mu.Lock()
	r.sawOutput = r.turnSeq
	if msg.SessionID != "" {
		r.sessionID = msg.SessionID
	}
	var toIdle bool
	if msg.Kind == protocol.KindResult {
		// Clamp: a result without a matching prompt (e.g. after an interrupt, Q-7) must not go negative.
		// Chặn dưới 0: result không khớp prompt nào (ví dụ sau interrupt, Q-7) không được làm âm.
		if r.pending > 0 {
			r.pending--
		}
		if r.pending == 0 && r.state == StateRunning {
			toIdle = r.setStateLocked(StateIdle)
		}
	}
	r.mu.Unlock()

	if msg.DecodeError != nil {
		r.log.Warn("unparsed stream line", slog.Int("bytes", len(msg.Raw)))
	}
	r.emit(Event{Type: EventMessage, Message: msg})
	if toIdle {
		r.emit(Event{Type: EventState, State: StateIdle})
	}
	if msg.Kind == protocol.KindAPIRetry && msg.Retry != nil && msg.Retry.Error == "authentication_failed" {
		r.fail("authentication failed: run `claude /login` with your subscription account")
	}
}

func (r *Runner) readStderr(stderr io.Reader) {
	sc := bufio.NewScanner(stderr)
	sc.Buffer(make([]byte, 0, 4096), 1<<20)
	for sc.Scan() {
		// Stderr can echo prompts; log only its size at debug. / Stderr có thể lặp lại prompt; chỉ log kích thước ở debug.
		r.log.Debug("agent stderr", slog.Int("bytes", len(sc.Bytes())))
	}
}

func (r *Runner) fail(reason string) {
	r.mu.Lock()
	changed := r.setStateLocked(StateFailed)
	r.mu.Unlock()
	if !changed {
		return
	}
	r.log.Error("agent failed", slog.String("reason", reason))
	r.emit(Event{Type: EventState, State: StateFailed, Reason: reason})
	_ = r.terminate()
}

func (r *Runner) finish(waitErr error) {
	// Reap commands the agent's tools left behind; the group outlives its leader (spike T12).
	// Dọn các lệnh mà tool của agent bỏ lại; group sống lâu hơn leader của nó (spike T12).
	if err := r.killGroup(); err != nil {
		r.log.Warn("kill leftover process group", slog.Any("error", err))
	}
	code := 0
	var exitErr *exec.ExitError
	switch {
	case waitErr == nil:
	case errors.As(waitErr, &exitErr):
		code = exitErr.ExitCode()
	default:
		code = -1
	}
	r.mu.Lock()
	r.exit = code
	if r.state != StateFailed {
		r.state = StateStopped
	}
	final := r.state
	r.mu.Unlock()

	r.log.Info("agent process exited", slog.Int("exit_code", code), slog.String("state", string(final)))
	r.emit(Event{Type: EventExited, State: final, ExitCode: code})
	r.emitMu.Lock()
	r.closed = true
	close(r.events)
	r.emitMu.Unlock()
	close(r.done)
}

// setStateLocked must be called with mu held; it never leaves a terminal state.
// setStateLocked phải được gọi khi đang giữ mu; không bao giờ rời trạng thái cuối.
func (r *Runner) setStateLocked(s State) bool {
	if r.state == s || r.state == StateFailed || r.state == StateStopped {
		return false
	}
	r.state = s
	return true
}

// emit delivers in order and drops nothing; after the process exits it becomes a no-op.
// emit giao event theo thứ tự, không làm rơi; sau khi process thoát thì không làm gì.
func (r *Runner) emit(e Event) {
	e.AgentID = r.cfg.AgentID
	e.At = time.Now().UTC()
	r.emitMu.Lock()
	defer r.emitMu.Unlock()
	if r.closed {
		return
	}
	r.events <- e
}

// SandboxDir resolves dir to an absolute, existing directory.
// SandboxDir chuyển dir thành đường dẫn tuyệt đối của một thư mục đang tồn tại.
func SandboxDir(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve %q: %w", dir, err)
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("stat %q: %w", abs, err)
	}
	if !fi.IsDir() {
		return "", fmt.Errorf("%q is not a directory", abs)
	}
	return abs, nil
}
