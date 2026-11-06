# Requirements — task-dispatch

Scope: be (orchestrator)
Source: design doc dự án nhóm "Giao việc" (2026-11-01); danh sách feature MVP đã duyệt (2026-11-05)
Code: chưa có
Mode: Prospective

Owner giao việc (task) cho một agent trên một project, gửi thêm tin trong task, xem agent làm realtime, ngắt turn đang chạy, huỷ hoặc đóng task. Giao việc = tạo worktree ([workspace](../workspace/requirements.md)) rồi đưa item `task` vào mailbox ([agent-mailbox](../agent-mailbox/requirements.md)). Stream đi qua kênh `/events` ([event-schema](../event-schema/requirements.md)).

## Glossary

| Term | Nghĩa |
|---|---|
| Task | Một việc owner giao cho một agent trên một project, có worktree riêng |
| Interrupt | Dừng turn đang chạy của task; task vẫn mở, tin sau tiếp tục cùng session (`--resume`) |
| Cancel | Kết thúc task; mọi item còn xếp hàng của task bị huỷ |

## User stories

- US-003 (agent-runtime): > As a **user**, I want **giao việc cho một agent và xem nó làm realtime**, so that **tôi điều khiển agent từ một chỗ**.
- US-018: > As a **user**, I want **dừng agent khi thấy nó đi sai hướng rồi nói lại cho nó làm tiếp**, so that **không phải giao lại từ đầu**.

## Acceptance criteria

### Tạo task

- R-1: WHEN client gọi `POST /tasks` với `project_id`, `agent_id`, `title` (1…120 ký tự), `prompt` (1…32768 ký tự), `parent_id` tùy chọn và header `Idempotency-Key` tùy chọn, THE SYSTEM SHALL tạo worktree cho task, rồi trong một transaction lưu task `queued`, đưa item `task` vào mailbox agent, ghi event `task.created`, và trả 201 với task gồm `worktree`. (platform)
- R-2: IF `Idempotency-Key` đã dùng cho một task, THEN THE SYSTEM SHALL trả 200 với task cũ, không tạo worktree hay item mới. (platform)
- R-3: IF project không `ready`, agent không `active`/`paused`, `parent_id` không tồn tại, hoặc field không hợp lệ, THEN THE SYSTEM SHALL trả lỗi tương ứng (409 `project_not_ready`, 409 `agent_unavailable`, 400 `invalid_argument`) và không tạo gì. (platform)
- R-4: IF tạo worktree thành công nhưng lưu task lỗi (mailbox đầy, DB lỗi), THEN THE SYSTEM SHALL trả lỗi và để worktree lại trên đĩa (không tự xoá, workspace R-11); lần tạo sau dùng tên có hậu tố. (platform)

### Trạng thái theo turn

- R-5: WHEN item của task được giao (`mailbox.delivered`), THE SYSTEM SHALL chuyển task sang `running`. (platform)
- R-6: WHEN turn của task kết thúc (`mailbox.done`) và task không còn item `queued` hay `delivering`, THE SYSTEM SHALL chuyển task sang `idle`, cộng `cost_usd`, `num_turns` của turn vào task, và lưu 2000 ký tự đầu của `result` làm `last_result_preview`. (platform)
- R-7: IF item của task `failed` (mailbox R-16), THEN THE SYSTEM SHALL chuyển task sang `failed` với lý do. (platform)

### Tin nhắn trong task

- R-8: WHEN client gọi `POST /tasks/{id}/messages` với `text` (1…32768) và `Idempotency-Key` tùy chọn cho task `queued`/`running`/`idle`, THE SYSTEM SHALL đưa item `user` có `task_id` vào mailbox agent và trả 202; task `done`/`cancelled`/`failed` trả 409 `task_closed`. (platform)

### Ngắt, huỷ, đóng

- R-9: WHEN client gọi `POST /tasks/{id}/interrupt` khi task `running`, THE SYSTEM SHALL yêu cầu agent-supervisor dừng turn (kill process group), đánh dấu item đang giao là `done` với ghi chú `interrupted` (không giao lại), chuyển task về `idle` trong vòng 5 giây, và ghi event; tin sau của owner chạy tiếp cùng session. Task không `running` trả 409 `not_running`. (platform)
- R-10: WHEN client gọi `POST /tasks/{id}/cancel`, THE SYSTEM SHALL dừng turn nếu đang chạy như R-9, huỷ mọi item `queued` của task (mailbox R-18), chuyển task sang `cancelled`, ghi event; worktree giữ nguyên. (platform)
- R-11: WHEN client gọi `POST /tasks/{id}/complete` cho task `idle`, THE SYSTEM SHALL chuyển task sang `done` và ghi event; task khác `idle` trả 409 `invalid_state`. (platform)

### Đọc

- R-12: WHEN client gọi `GET /tasks?agent_id=&project_id=&status=`, THE SYSTEM SHALL trả danh sách task mới nhất trước, không kèm `prompt`; `GET /tasks/{id}` trả đầy đủ kèm `prompt`, worktree, chi phí, task con. (platform)
- R-13: WHILE client nghe `GET /events?task_id=<id>`, THE SYSTEM SHALL gửi mọi event của task đó (turn, tool, approval của agent trong turn của task, mailbox, task) theo event-schema R-11…R-13. (platform)

## Security

- SEC-1: THE SYSTEM SHALL không log `prompt`, `text` của tin nhắn hay `result`.
- SEC-2: Mọi endpoint ghi chỉ nhận từ client đã xác thực; tới khi có, HTTP bind loopback (approval-inbox#TD-3).

## Contract surface

| Kind | Name | Owner | Notes |
|---|---|---|---|
| HTTP | `POST /tasks`, `GET /tasks`, `GET /tasks/{id}`, `POST /tasks/{id}/messages`, `POST /tasks/{id}/interrupt`, `POST /tasks/{id}/cancel`, `POST /tasks/{id}/complete` | this feature | header `Idempotency-Key` |
| Event | `task.created`, `task.status_changed` | this feature | |
| Table | `tasks` | this feature | |
| Go | `workspace.Service.CreateWorktree`, `mailbox.Service.Enqueue/CancelQueued`, `supervisor.Service.Stop` | features khác | |

## Out of scope

- Board kéo thả, gán lại assignee, log tổng hợp: roadmap `task-board`.
- Agent tự tạo task con: chưa có; `parent_id` chỉ do owner đặt.
- Tự động đóng task khi agent "xong": owner đóng bằng R-11.

## Open questions

| ID | Câu hỏi | Owner | Liên quan |
|---|---|---|---|
