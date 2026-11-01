# Requirements — approval-inbox

Scope: be (orchestrator)
Source: design doc dự án (project memory, 2026-11-01), nhóm tính năng "Duyệt"
Code: `internal/domain/approval.go`, `internal/usecase/approval/`, `internal/adapter/repository/memory/`
Mode: Mixed (domain + use case: Retroactive; delivery HTTP, push, MCP: Prospective)

Inbox duyệt tập trung cho mọi agent: agent xin chạy tool hoặc chủ động hỏi, request nằm ở trạng thái pending tới khi user quyết định hoặc quá deadline. Quá deadline thì luôn là deny. Cách agent gửi request vào inbox (MCP permission prompt tool) thuộc [agent-runtime](../agent-runtime/requirements.md).

## Glossary

| Term | Nghĩa |
|---|---|
| Approval | Một yêu cầu duyệt, kind `tool` (xin chạy tool) hoặc `ask` (agent hỏi user) |
| Deadline | `created_at + ORCH_APPROVAL_TIMEOUT` |
| Default deny | Hết deadline mà chưa có quyết định thì xử lý như deny |

## User stories

- US-001: > As a **user điều khiển nhiều agent**, I want **mọi yêu cầu duyệt dồn về một inbox**, so that **tôi không phải canh từng terminal**.
- US-002: > As a **user**, I want **yêu cầu không ai trả lời tự bị từ chối**, so that **agent không bao giờ tự làm việc nguy hiểm khi tôi vắng mặt**.

## Acceptance criteria

### Mở yêu cầu

- R-1: WHEN một agent gửi yêu cầu duyệt hợp lệ, THE SYSTEM SHALL lưu approval với status `pending` và deadline = thời điểm tạo + `ORCH_APPROVAL_TIMEOUT` (mặc định 5m, tối thiểu 10s). (platform)
- R-2: IF yêu cầu thiếu `agent_id`, kind không thuộc `tool`/`ask`, kind `tool` thiếu tên tool, risk không thuộc `low`/`high`, hoặc input không phải JSON hợp lệ, THEN THE SYSTEM SHALL từ chối với lỗi `invalid argument` liệt kê mọi vấn đề cùng lúc và không lưu gì. (platform)
- R-3: THE SYSTEM SHALL lưu bản copy của input, để thay đổi của caller sau khi gửi không làm đổi nội dung đã lưu. (platform)

### Quyết định

- R-4: WHEN user gửi `allow` hoặc `deny` trước deadline cho một approval `pending`, THE SYSTEM SHALL chuyển status sang `allowed` / `denied`, ghi `decided_by`, `decided_at`, `reason`. (platform)
- R-5: IF quyết định đến đúng hoặc sau deadline, THEN THE SYSTEM SHALL lưu approval ở status `expired` và trả lỗi `late decision`. Agent không được tiếp tục. (platform)
- R-6: IF approval không còn `pending`, THEN THE SYSTEM SHALL trả lỗi `approval already decided` và giữ nguyên quyết định cũ. (platform)
- R-7: WHEN nhiều quyết định cho cùng một approval đến đồng thời, THE SYSTEM SHALL chấp nhận đúng một quyết định, các quyết định còn lại nhận `approval already decided`. (platform)
- R-8: IF quyết định thiếu người duyệt hoặc giá trị không phải `allow`/`deny`, THEN THE SYSTEM SHALL trả `invalid argument` và không đổi trạng thái. (platform)
- R-9: IF approval id không tồn tại, THEN THE SYSTEM SHALL trả `not found`. (platform)

### Hết hạn

- R-10: WHILE orchestrator đang chạy, THE SYSTEM SHALL quét mỗi 1 giây và chuyển mọi approval `pending` đã tới deadline sang `expired`, với `decided_by = system`. (platform)
- R-11: IF một bản ghi lỗi khi quét, THEN THE SYSTEM SHALL tiếp tục với các bản ghi còn lại và log lỗi. (platform)

### Inbox cho user (Prospective)

- R-12: WHEN user mở inbox, THE SYSTEM SHALL trả danh sách approval `pending` của mọi agent, deadline gần nhất trước, có agent, kind, tool, risk, deadline.
  > ⚠️ Not confirmed: shape của API inbox (REST hay chỉ WebSocket event) — chặn delivery handler. (Q-1)
- R-13: WHEN có approval mới với risk `high`, THE SYSTEM SHALL gửi push notification tới thiết bị của user.
  > ⚠️ Not confirmed: kênh push GĐ1 (Web Push, Telegram, hay cả hai) — chặn adapter notify. (Q-2)

## Security

- SEC-1: THE SYSTEM SHALL không log tool input của approval (có thể chứa nội dung file hoặc secret). Log chỉ gồm id, agent, kind, tool, risk, deadline, status.
- SEC-2: THE SYSTEM SHALL không có lựa chọn "always allow" khi duyệt từ xa; mỗi approval quyết định riêng. (platform)
- SEC-3: WHEN quyết định được ghi, THE SYSTEM SHALL lưu người duyệt và thời điểm để audit. Audit log bất biến thuộc TD-2.
- SEC-4: THE SYSTEM SHALL chỉ nhận quyết định từ client đã xác thực. Mô hình xác thực web thuộc feature khác; tới khi có, HTTP bind loopback (TD-3).

## Contract surface

| Kind | Name | Owner | Notes |
|---|---|---|---|
| Env var | `ORCH_APPROVAL_TIMEOUT` | this feature | Go duration, ≥ 10s, default 5m |
| Go port | `port.ApprovalRepository` | this feature | `Update` phải atomic |
| HTTP | inbox list / decide | this feature | ⚠️ chưa thiết kế (Q-1) |
| MCP tool | `approval_prompt` | [agent-runtime](../agent-runtime/requirements.md) | đầu vào của R-1 |

## Out of scope

- Agent gửi yêu cầu vào inbox qua MCP: [agent-runtime](../agent-runtime/requirements.md).
- Lưu bền (SQLite/Postgres): TD-1.
- Đánh dấu risk tự động theo pattern lệnh: feature sau, hiện agent-runtime truyền risk vào.

## Open questions

| ID | Câu hỏi | Owner | Liên quan |
|---|---|---|---|
| Q-1 | API inbox: REST `GET /approvals` + `POST /approvals/{id}/decision`, hay chỉ qua WebSocket? Đề xuất: REST + event qua SSE | Đức | R-12 |
| Q-2 | Kênh push GĐ1? Đề xuất: Telegram trước (không cần HTTPS public) | Đức | R-13 |
