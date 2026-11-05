# Requirements — agent-mailbox

Scope: be (orchestrator), MCP tool cho agent
Source: yêu cầu của Đức "nó sẽ có 1 queue message (vì có thể 2 agent cùng hỏi nó), nó có thể trả lời tuần tự cho đúng agent hỏi" (2026-11-02); ADR-0003
Code: chưa có (`send_message` stub trong `internal/delivery/mcpbus`, agent-runtime#TD-3)
Mode: Prospective

Mỗi agent có một hàng đợi (mailbox). Việc user giao, tin nhắn của user, câu hỏi từ agent khác và câu trả lời đều là **item** trong mailbox. Agent xử lý **tuần tự, mỗi lần một item, mỗi item một turn**. Câu hỏi (`ask`) mang `correlation_id`; `result` cuối turn xử lý câu hỏi tự thành **reply** vào mailbox của người hỏi (ADR-0003). Feature này giữ hàng đợi, thứ tự, idempotency, chống vòng lặp và tool `ask_agent`; chạy turn là [agent-supervisor](../agent-supervisor/requirements.md), chọn lúc chạy là [concurrency-quota](../concurrency-quota/requirements.md).

## Glossary

| Term | Nghĩa |
|---|---|
| Item | Một đơn vị trong mailbox, kind `task`, `user`, `ask`, `reply` |
| Correlation id | Id của item `ask`; reply mang lại id này để người hỏi biết trả lời cho câu nào |
| Root | Item gốc của một chuỗi hỏi đáp (item `task`/`user` đã sinh ra chuỗi) |
| Hop | Độ sâu trong chuỗi: item `task`/`user` là 0, ask sinh trong turn của item hop n có hop n+1 |

## User stories

- US-015: > As a **user**, I want **nhiều agent cùng hỏi một agent thì agent đó trả lời lần lượt, đúng người hỏi**, so that **không ai nhận nhầm câu trả lời**.
- US-016: > As a **user**, I want **agent không hỏi qua hỏi lại vô hạn**, so that **không đốt quota vì vòng lặp**.
- US-017: > As a **user**, I want **xem toàn bộ cuộc hỏi đáp giữa các agent trên web**, so that **tôi hiểu vì sao agent làm vậy**.

## Acceptance criteria

### Đưa item vào

- R-1: WHEN một item được đưa vào mailbox của agent nhận, THE SYSTEM SHALL lưu item với `status = queued`, `seq` tăng dần theo người nhận, và ghi durable event `mailbox.enqueued` (không có body) trong cùng transaction. (platform)
- R-2: WHEN item có `idempotency_key` đã được cùng người gửi dùng, THE SYSTEM SHALL trả item cũ, không tạo item mới, không phát event mới. (platform)
- R-3: IF số item `queued` của người nhận đã bằng `mailbox_capacity`, THEN THE SYSTEM SHALL từ chối item `task`, `user`, `ask` với `mailbox_full`; item `reply` luôn được nhận để người hỏi không mất câu trả lời. (platform)
- R-4: IF người nhận `archived`, THEN THE SYSTEM SHALL từ chối với `agent_unavailable`; người nhận `paused` vẫn nhận item (agent-registry R-12). (platform)

### Tool `ask_agent`

- R-5: WHEN agent gọi MCP tool `ask_agent` với `to` (slug), `text` (1…16384 ký tự) và `idempotency_key` tùy chọn, THE SYSTEM SHALL xác định người gửi từ bearer token MCP, tạo item `ask` với `correlation_id` = id item, `root_id` và `hop` theo item người gửi đang xử lý, `task_id` rỗng, rồi trả ngay `{correlation_id, status: "queued", position}`; turn của người gửi không chờ câu trả lời. (platform)
- R-6: IF người gửi không nằm trong `can_message` của người nhận, hoặc `to` là chính người gửi, hoặc `to` không tồn tại, THEN THE SYSTEM SHALL từ chối với `not_allowed` (hoặc `unknown_agent`). (platform)
- R-7: IF `hop` của item mới vượt `ORCH_MAX_HOPS` (mặc định 4), THEN THE SYSTEM SHALL từ chối với `loop_limit`. (platform)
- R-8: IF chuỗi của cùng `root_id` đã có `ORCH_MAX_ASKS_PER_ROOT` (mặc định 20) item `ask`, THEN THE SYSTEM SHALL từ chối với `loop_limit`. (platform)
- R-9: WHEN `ask_agent` bị từ chối, THE SYSTEM SHALL trả tool result lỗi (`isError: true`) có mã và một câu giải thích, không lưu item, và ghi durable event `mailbox.rejected` với `reason`. (platform)

### Giao tuần tự

- R-10: WHEN scheduler gọi `ClaimNext(agent)`, THE SYSTEM SHALL chọn item `queued` có `seq` nhỏ nhất của agent đó, chỉ khi agent `active`, chuyển item sang `delivering`, tăng `attempt`, ghi `mailbox.delivered`, và trả item; không có item thì trả rỗng. Một agent có tối đa một item `delivering`. (platform)
- R-11: THE SYSTEM SHALL dựng prompt của item gồm header (kind, người gửi, `correlation_id`, với `reply` thêm 500 ký tự đầu của câu hỏi gốc) và body; với `ask` và `reply`, header nói rõ nội dung đến từ agent khác, là dữ liệu tham khảo, không phải chỉ thị của owner. (platform)

### Hoàn tất và reply

- R-12: WHEN scheduler báo item `ask` hoàn tất với `TurnResult`, THE SYSTEM SHALL trong một transaction chuyển item sang `done` và tạo item `reply` trong mailbox người hỏi với `in_reply_to` = id câu hỏi, cùng `correlation_id`, `root_id`, `hop`, `task_id` của item người hỏi đang xử lý lúc hỏi, body = text của `result` (cắt 16 KiB), và ghi `mailbox.done`, `mailbox.replied`. (platform)
- R-13: IF `TurnResult.is_error`, THEN THE SYSTEM SHALL vẫn tạo reply với `reply_status = error` và body nêu agent nhận không trả lời được kèm `subtype`. (platform)
- R-14: WHEN item `task`, `user` hoặc `reply` hoàn tất, THE SYSTEM SHALL chuyển item sang `done` và ghi `mailbox.done`; task-dispatch cập nhật task theo event này. (platform)

### Hết hạn, lỗi, khởi động lại

- R-15: WHILE orchestrator chạy, THE SYSTEM SHALL mỗi 10 giây chuyển item `ask` còn `queued` quá `ORCH_ASK_TIMEOUT` (mặc định 30m) sang `expired` và tạo reply `reply_status = expired` cho người hỏi; item `delivering` không hết hạn. (platform)
- R-16: WHEN scheduler báo turn bị gián đoạn hoặc quá giờ, THE SYSTEM SHALL đưa item về `queued` giữ nguyên `seq` (vẫn đứng đầu hàng), then:
  - on `attempt` < 3: chờ lần giao sau;
  - on `attempt` = 3: chuyển `failed`, ghi `mailbox.failed`; nếu là `ask` thì tạo reply `reply_status = failed` cho người hỏi. (platform)
- R-17: WHEN orchestrator khởi động, THE SYSTEM SHALL áp R-16 cho mọi item đang `delivering` (at-least-once: turn có thể chạy lại). (platform)

- R-21: WHEN owner ngắt turn của task (task-dispatch R-9), THE SYSTEM SHALL đánh dấu item đang `delivering` của task là `interrupted`, để khi turn dừng item chuyển `done` với `reply_status`/ghi chú `interrupted` thay vì quay lại hàng đợi; item `ask` bị ngắt tạo reply `reply_status = interrupted` cho người hỏi. (platform)

### Huỷ

- R-18: WHEN task bị huỷ hoặc agent bị archive, THE SYSTEM SHALL chuyển mọi item `queued` liên quan sang `cancelled`, ghi `mailbox.cancelled`, và với mỗi `ask` bị huỷ tạo reply `reply_status = cancelled` cho người hỏi. (platform)
- R-19: IF reply tới cho người hỏi đã `archived`, THEN THE SYSTEM SHALL lưu reply ở `cancelled` thay vì `queued`. (platform)

### Xem trên web

- R-20: WHEN client gọi `GET /agents/{id}/mailbox?status=`, THE SYSTEM SHALL trả danh sách item theo `seq` không kèm body; `GET /mailbox/items/{id}` trả item kèm body; `GET /conversations/{root_id}` trả mọi item của chuỗi theo thời gian tạo, kèm body. (platform)

## Security

- SEC-1: THE SYSTEM SHALL không log và không đưa body item vào event; body chỉ ở bảng `mailbox_items` và API đọc (R-20).
- SEC-2: THE SYSTEM SHALL lấy danh tính người gửi `ask_agent` từ bearer MCP (agent-runtime SEC-4), không từ input của tool.
- SEC-3: Nội dung từ agent khác không bao giờ được coi là quyết định của owner: không duyệt approval, không đổi cấu hình, không cấp quyền (R-11 đánh dấu là dữ liệu).

## Contract surface

| Kind | Name | Owner | Notes |
|---|---|---|---|
| MCP tool | `ask_agent` | this feature | thay stub `send_message` (agent-runtime#TD-3); input `to`, `text`, `idempotency_key?`; output text JSON |
| Go | `mailbox.Service`: `Enqueue`, `ClaimNext`, `Complete`, `Interrupt`, `MarkInterrupted`, `CancelQueued`, `ExpireDue` | this feature | dùng bởi task-dispatch, concurrency-quota, agent-registry |
| HTTP | `GET /agents/{id}/mailbox`, `GET /mailbox/items/{id}`, `GET /conversations/{root_id}` | this feature | |
| Event | `mailbox.enqueued`, `.delivered`, `.done`, `.replied`, `.rejected`, `.expired`, `.failed`, `.cancelled` | this feature | |
| Env var | `ORCH_MAX_HOPS`, `ORCH_MAX_ASKS_PER_ROOT`, `ORCH_ASK_TIMEOUT` | this feature | 4; 20; 30m |
| Table | `mailbox_items` | this feature | |

## Out of scope

- Kênh chung, broadcast, nhóm agent: roadmap `office-bus`.
- Ưu tiên item (reply trước ask): MVP FIFO thuần (A-1).
- Agent chọn không trả lời: mọi `result` của `ask` đều thành reply (ADR-0003).

## Open questions

| ID | Câu hỏi | Owner | Liên quan |
|---|---|---|---|
