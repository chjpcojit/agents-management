# Requirements — agent-supervisor

Scope: be (orchestrator), process `claude` trên máy owner
Source: agent-runtime v1.5 (spike xong T1–T15), agent-runtime#TD-4; danh sách feature MVP đã duyệt (2026-11-05)
Code: chưa có (`internal/adapter/runtime/claudecli` là spike-grade)
Mode: Prospective

Supervisor biến hồ sơ agent ([agent-registry](../agent-registry/requirements.md)) và quyền ([data-access](../data-access/requirements.md)) thành process `claude -p` thật, chạy **một turn cho một item** khi scheduler giao, giữ `session_id` để `--resume`, phát event của turn, và dừng / khởi động lại process khi cần. Cách nói chuyện với một process (stream-json, approval MCP, kill group) đã chốt ở [agent-runtime](../agent-runtime/requirements.md); feature này thêm port `AgentRuntime` và vòng đời theo agent.

## Glossary

| Term | Nghĩa |
|---|---|
| Scope | Ngữ cảnh chạy của một session: worktree của một task, hoặc thư mục home của agent |
| Session | Cặp (agent, scope) ↔ `session_id` của Claude Code; transcript gắn với `cwd` nên mỗi scope một session |
| Turn | Một lần gửi một item và nhận đúng một `result` |
| Runtime state | `offline`, `starting`, `idle`, `running`, `stopping`, `failed` |

## User stories

- US-003 (agent-runtime): > As a **user**, I want **giao việc cho một agent và xem nó làm realtime**, so that **tôi điều khiển agent từ một chỗ**.
- US-012: > As a **user**, I want **agent tự chạy lại và nhớ hội thoại cũ khi process chết hay máy khởi động lại**, so that **tôi không phải giao lại từ đầu**.

## Acceptance criteria

### Chạy một turn

- R-1: WHEN scheduler gọi `RunTurn(agent, item)`, THE SYSTEM SHALL xác định scope của item theo [design](design.md#chọn-scope), bảo đảm có process `idle` của agent cho đúng scope đó (spawn nếu cần), gửi đúng một prompt, chuyển runtime state sang `running`, và trả về khi nhận `result` của prompt đó. (platform)
- R-2: THE SYSTEM SHALL không bao giờ gửi prompt thứ hai cho một process khi prompt trước chưa có `result` trên đường của scheduler (ADR-0003). (platform)
- R-3: WHEN spawn, THE SYSTEM SHALL dựng lệnh theo bảng `LaunchSpec` ở [design](design.md#launchspec) từ cấu hình agent, `AccessPlan` và scope, thêm `--resume <session_id>` nếu scope đã có session. (platform)
- R-4: WHEN nhận `system/init` hoặc `result`, THE SYSTEM SHALL lưu `session_id` cho (agent, scope) để lần sau resume. (platform)
- R-5: WHILE turn chạy, THE SYSTEM SHALL phát event `turn.started`, `turn.text`, `tool.requested` (kèm resources), `tool.input`, `tool.completed`, `tool.output`, `turn.completed` theo [event-schema](../event-schema/requirements.md), gắn `agent_id`, `task_id` của item, `session_id`, và `correlation_id` của item. (platform)
- R-6: WHEN turn kết thúc, THE SYSTEM SHALL trả cho scheduler `TurnResult{text, is_error, subtype, cost_usd, usage, duration_ms}`; text là `result` cuối turn (dùng làm reply theo ADR-0003). (platform)

### Process dùng lại và hết hạn

- R-7: WHEN item tiếp theo cùng scope với process đang `idle` và `config_version` lẫn `access_version` của agent không đổi từ lúc spawn, THE SYSTEM SHALL dùng lại process đó. (platform)
- R-8: WHEN item tiếp theo khác scope, hoặc cấu hình / quyền của agent đã đổi, THE SYSTEM SHALL đóng process `idle` hiện tại (đóng stdin, chờ tối đa 10s, rồi kill group) và spawn process mới cho scope của item với `--resume` nếu có. (platform)
- R-9: WHEN process ở `idle` quá `ORCH_AGENT_IDLE_TTL` (mặc định 10m), THE SYSTEM SHALL đóng nó và chuyển agent về `offline`; session vẫn được giữ. (platform)

### Lỗi và dừng

- R-10: IF process thoát khi đang `running`, THEN THE SYSTEM SHALL kill phần còn lại của process group, phát `agent.exited`, chuyển agent `offline`, và trả `ErrTurnInterrupted` cho scheduler để item được giao lại (agent-mailbox R-16). (platform)
- R-11: IF không có dòng stdout trong 90 giây sau spawn hoặc sau gửi prompt, hoặc có `system/api_retry` với `error = authentication_failed`, THEN THE SYSTEM SHALL dừng process, chuyển agent `failed` với lý do, và trả `ErrAgentFailed`; agent `failed` không nhận turn mới tới khi owner gọi `POST /agents/{id}/runtime/reset`. (platform)
- R-12: IF turn chạy quá `ORCH_TURN_TIMEOUT` (mặc định 60m), THEN THE SYSTEM SHALL dừng process như huỷ turn và trả `ErrTurnTimeout`. (platform)
- R-13: WHEN được yêu cầu dừng (huỷ turn từ task-dispatch, thu hồi quyền từ data-access, archive), THE SYSTEM SHALL kill process group (SIGTERM, sau 3s SIGKILL; agent-runtime R-12), chuyển agent `offline` trong vòng 5 giây, và trả `ErrTurnInterrupted` cho turn đang chờ nếu có. (platform)
- R-14: WHEN agent bị `paused`, THE SYSTEM SHALL để turn đang chạy chạy nốt và không nhận turn mới; WHEN agent bị `archived`, THE SYSTEM SHALL đóng process `idle` của nó. (platform)
- R-15: WHEN orchestrator khởi động, THE SYSTEM SHALL coi mọi agent là `offline` (process không sống qua restart) và giữ mọi session đã lưu. (platform)

### Quan sát và điều khiển

- R-16: WHEN runtime state đổi, THE SYSTEM SHALL ghi durable event `agent.runtime_changed` với `state` và `reason`. (platform)
- R-17: WHEN client gọi `GET /agents/{id}/runtime`, THE SYSTEM SHALL trả 200 với `state`, `scope`, `session_id`, `config_version` và `access_version` của process đang chạy, thời điểm turn hiện tại bắt đầu. (platform)
- R-18: WHEN client gọi `POST /agents/{id}/runtime/reset` cho agent `failed`, THE SYSTEM SHALL chuyển về `offline` để scheduler giao tiếp; agent không `failed` trả 409 `invalid_state`. (platform)

## Security

- SEC-1: THE SYSTEM SHALL spawn với env allowlist, settings deny `.claude/**`, base policy ask, `--strict-mcp-config` như agent-runtime R-2, SEC-2, SEC-5.
- SEC-2: THE SYSTEM SHALL cấp bearer token MCP mới (256 bit) cho mỗi lần spawn và thu hồi khi process dừng (agent-runtime SEC-4).
- SEC-3: THE SYSTEM SHALL không log prompt, text, tool input/output (CLAUDE.md rule 5); `--append-system-prompt` và `--settings` được truyền qua file tạm quyền `0600` xoá khi process dừng, không qua argv, để `ps` không lộ prompt.

## Contract surface

| Kind | Name | Owner | Notes |
|---|---|---|---|
| Go port | `port.AgentRuntime`, `port.RuntimeProcess`, `port.RuntimeEvent` | this feature | resolve agent-runtime#TD-4 |
| Go port | `port.AgentStopper`, `port.RuntimeStatus` | this feature | dùng bởi data-access, agent-registry, task-dispatch |
| Go | `supervisor.Service.RunTurn` | this feature | gọi bởi [concurrency-quota](../concurrency-quota/requirements.md) |
| HTTP | `GET /agents/{id}/runtime`, `POST /agents/{id}/runtime/reset` | this feature | |
| Event | `agent.runtime_changed`, `agent.exited`, `turn.*`, `tool.*`, `stream.unknown` | this feature | |
| Env var | `ORCH_AGENT_IDLE_TTL`, `ORCH_TURN_TIMEOUT`, `ORCH_CLAUDE_BIN` | this feature | 10m; 60m; `claude` |
| Table | `agent_sessions` | this feature | |
| CLI | `claude -p` flags | Anthropic | xem agent-runtime |

## Out of scope

- Chọn agent nào chạy và giới hạn đồng thời: [concurrency-quota](../concurrency-quota/requirements.md).
- Hàng đợi item và reply: [agent-mailbox](../agent-mailbox/requirements.md).
- Tạo worktree: [workspace](../workspace/requirements.md).

## Open questions

| ID | Câu hỏi | Owner | Liên quan |
|---|---|---|---|
| Q-1 | Với stdin stream-json giữ mở, `--max-turns` giới hạn mỗi prompt hay cả process? | spike trên máy Đức | `max_turns_per_task`, A-1 |
