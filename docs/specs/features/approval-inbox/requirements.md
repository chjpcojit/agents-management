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

- R-12: WHEN client gọi `GET /approvals?status=pending`, THE SYSTEM SHALL trả 200 với danh sách approval `pending` của mọi agent, deadline gần nhất trước, mỗi phần tử có `id`, `agent_id`, `kind`, `tool_name`, `risk`, `created_at`, `deadline`. Tool input không nằm trong danh sách. (platform)
- ~~R-13: WHEN có approval mới với risk `high`, THE SYSTEM SHALL gửi push notification tới thiết bị của user.~~ Removed in v1.2: Đức hoãn push notification, GĐ1 chỉ duyệt trên web qua SSE (R-16…R-18); xem TD-4.
- R-14: WHEN client gọi `GET /approvals/{id}`, THE SYSTEM SHALL trả 200 với approval đầy đủ, kể cả tool input, để người duyệt xem trước khi quyết định; id không tồn tại trả 404. (platform)
- R-15: WHEN client gọi `POST /approvals/{id}/decision` với `decision` (`allow`/`deny`) và `reason` tùy chọn, THE SYSTEM SHALL áp dụng R-4…R-9 và trả, then:
  - on success: 200 với approval sau khi quyết định;
  - on failure (body sai hoặc decision lạ): 400;
  - on failure (không tồn tại): 404;
  - on failure (đã quyết định): 409 `already_decided`;
  - on failure (quá hạn): 409 `expired`, approval đã lưu `expired`. (platform)
- R-16: WHILE một client giữ kết nối `GET /events` (SSE), THE SYSTEM SHALL đẩy event `approval.requested`, `approval.decided`, `approval.expired` trong vòng 1 giây kể từ khi trạng thái đổi, mỗi event có `id` tăng dần và payload như một phần tử của R-12 cộng `status`. (platform)
- R-17: WHEN client SSE kết nối lại với header `Last-Event-ID`, THE SYSTEM SHALL gửi lại các event sau id đó còn trong buffer 1000 event gần nhất; nếu id đã rơi khỏi buffer, THE SYSTEM SHALL gửi event `resync` để client gọi lại R-12. (platform)
- R-18: THE SYSTEM SHALL gửi comment SSE heartbeat mỗi 15 giây để proxy và trình duyệt không đóng kết nối rảnh. (platform)

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
| HTTP | `GET /approvals` | this feature | shape trong `api/openapi.yaml` (tạo ở Phase 4) |
| HTTP | `GET /approvals/{id}` | this feature | như trên |
| HTTP | `POST /approvals/{id}/decision` | this feature | như trên |
| SSE | `GET /events`: `approval.requested`, `approval.decided`, `approval.expired`, `resync` | this feature | kênh chung, stream tiến độ agent sẽ dùng lại |
| Error body | RFC 7807 problem (`type`, `title`, `status`, `code`) | this feature | `code` theo bảng error mapping |
| MCP tool | `approval_prompt` | [agent-runtime](../agent-runtime/requirements.md) | đầu vào của R-1 |

## Out of scope

- Agent gửi yêu cầu vào inbox qua MCP: [agent-runtime](../agent-runtime/requirements.md).
- Lưu bền (SQLite/Postgres): TD-1.
- Đánh dấu risk tự động theo pattern lệnh: feature sau; hiện `domain.ClassifyRisk` xếp theo tên tool (đọc là low, còn lại high).
- Push notification (Web Push, Telegram) và app mobile: hoãn, TD-4.

## Open questions

| ID | Câu hỏi | Owner | Liên quan |
|---|---|---|---|
