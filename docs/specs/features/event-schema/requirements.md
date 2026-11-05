# Requirements — event-schema

Scope: be (orchestrator), client web qua SSE
Source: danh sách feature MVP đã duyệt (Đức, 2026-11-05); ADR-0003
Code: chưa có
Mode: Prospective

Mọi thứ xảy ra trong orchestrator (agent đổi trạng thái, turn chạy, tool được gọi, approval, mailbox, task, cấp quyền) được phát thành **event** theo một envelope chung v0. Web nhận event qua một kênh SSE duy nhất, kết nối lại không mất event. Feature này định nghĩa envelope, danh mục type, cách gắn `resource` cho tool call, và kênh `/events`. Event của từng feature do feature đó phát; feature này không chứa rule nghiệp vụ của chúng.

## Glossary

| Term | Nghĩa |
|---|---|
| Event | Một bản ghi bất biến về việc đã xảy ra, theo envelope v0 |
| Durable event | Event không chứa nội dung (prompt, text, tool input, file); lưu Postgres, phát lại được sau restart |
| Live event | Event có nội dung (text agent viết, tool input); chỉ gửi qua SSE và ring buffer bộ nhớ, không lưu DB, không log |
| Resource | Nơi một tool call đụng vào: `{store, op, target, inferred}` |

## User stories

- US-005: > As a **user**, I want **thấy realtime mọi việc agent đang làm, kể cả agent đọc ghi dữ liệu nào**, so that **tôi phát hiện và dừng agent làm sai sớm**.
- US-006: > As a **user**, I want **web mất mạng rồi kết nối lại không bị hụt event**, so that **trạng thái trên web luôn đúng**.

## Acceptance criteria

### Envelope

- R-1: THE SYSTEM SHALL phát mọi event với envelope gồm `id` (số nguyên tăng dần toàn cục), `v` (phiên bản schema, `0`), `ts` (UTC, RFC 3339 có mili giây), `type`, `durable` (bool), và tùy theo event: `agent_id`, `task_id`, `session_id`, `correlation_id`, cùng `payload` (object). (platform)
- R-2: THE SYSTEM SHALL đặt tên `type` theo dạng `<nhóm>.<việc>` trong danh mục ở [design](design.md#danh-mục-type); type không có trong danh mục thì không được phát. (platform)
- R-3: THE SYSTEM SHALL bảo đảm event của cùng một agent có `id` tăng theo đúng thứ tự xảy ra, và thứ tự stdout của process `claude` được giữ nguyên. (platform)

### Nội dung và lưu trữ

- R-4: THE SYSTEM SHALL đánh dấu event chứa nội dung (text agent, tool input, tool output, body tin nhắn) là live (`durable = false`); live event không được ghi vào Postgres hay log. (platform)
- R-5: WHEN một use case đổi trạng thái được lưu (approval, mailbox, task, agent, grant), THE SYSTEM SHALL ghi durable event trong cùng transaction với thay đổi đó, rồi mới phát ra SSE sau khi commit. Transaction rollback thì không có event. (platform)
- R-6: IF ghi durable event thất bại, THEN THE SYSTEM SHALL rollback cả thay đổi trạng thái và trả lỗi cho caller. (platform)

### Resource của tool call

- R-7: WHEN agent gọi một tool, THE SYSTEM SHALL phát event `tool.requested` có `tool_name`, `tool_use_id` và danh sách `resources`, mỗi phần tử `{store, op, target, inferred}` với `store` thuộc `worktree`, `fs`, `git`, `github`, `web`, `source:<tên nguồn>`, `orchestrator`, `unknown` và `op` thuộc `read`, `write`, `delete`, `exec`, `network`. (platform)
- R-8: WHEN tool có input có cấu trúc (`Read`, `Write`, `Edit`, `Glob`, `Grep`, `WebFetch`, tool MCP của nguồn dữ liệu), THE SYSTEM SHALL suy ra resource từ field của input với `inferred = false`. (platform)
- R-9: WHEN tool là `Bash`, THE SYSTEM SHALL suy ra resource bằng heuristic trên lệnh với `inferred = true`; lệnh không nhận ra được cho ra đúng một resource `{store: unknown, op: exec, target: "", inferred: true}`. (platform)
- R-10: THE SYSTEM SHALL ghi `target` là đường dẫn tương đối với worktree (hoặc tuyệt đối nếu ngoài worktree), host của URL, hoặc tên tool MCP, tối đa 256 ký tự; không bao giờ là nguyên văn lệnh, query string hay giá trị input. (platform)

### Kênh SSE

- R-11: WHILE client giữ kết nối `GET /events`, THE SYSTEM SHALL gửi mỗi event trong vòng 1 giây kể từ khi phát, dạng SSE `id:`, `event: <type>`, `data: <envelope JSON>`. (platform)
- R-12: WHEN client gọi `GET /events` với query `agent_id` và/hoặc `task_id`, THE SYSTEM SHALL chỉ gửi event khớp mọi filter đã cho. (platform)
- R-13: WHEN client kết nối lại với header `Last-Event-ID`, THE SYSTEM SHALL gửi lại theo đúng thứ tự mọi durable event sau id đó từ Postgres và các live event sau id đó còn trong ring buffer, then:
  - on success: tiếp tục stream event mới;
  - on failure (live event sau id đã rơi khỏi buffer): gửi thêm event `stream.gap` có `from_id`, để client biết phần text bị hụt. (platform)
- R-14: THE SYSTEM SHALL gửi comment SSE heartbeat mỗi 15 giây. (platform)
- R-15: IF một subscriber không đọc kịp (hàng đợi gửi đầy 256 event), THEN THE SYSTEM SHALL đóng kết nối đó; client kết nối lại với `Last-Event-ID`. (platform)

### Tương thích

- R-16: THE SYSTEM SHALL chỉ thêm field hoặc type mới trong cùng `v`; đổi tên, đổi kiểu hoặc bỏ field là đổi `v` và ghi vào changelog. Client phải bỏ qua field và type không biết. (platform)
- R-17: IF dòng stdout của `claude` có `type` lạ, THEN THE SYSTEM SHALL phát live event `stream.unknown` với độ dài dòng, không kèm nội dung dòng. (platform)

## Security

- SEC-1: THE SYSTEM SHALL không ghi nội dung prompt, text, tool input/output, body tin nhắn vào log hay durable event (CLAUDE.md rule 5). Live event chỉ tới client đã kết nối.
- SEC-2: THE SYSTEM SHALL chỉ phục vụ `/events` trên địa chỉ HTTP của orchestrator (loopback, CLAUDE.md rule 8). Xác thực client web thuộc approval-inbox#TD-3.

## Contract surface

| Kind | Name | Owner | Notes |
|---|---|---|---|
| Schema | `api/events/v0.schema.json` | this feature | JSON Schema envelope + payload từng type |
| SSE | `GET /events?agent_id=&task_id=` | this feature | thay kênh SSE của approval-inbox R-16…R-18 |
| Table | `events` | this feature | tạo bằng migration của [storage](../storage/requirements.md) |
| Go port | `port.EventStore`, `port.EventPublisher` | this feature | |
| Env var | `ORCH_EVENT_BUFFER` | this feature | số live event giữ trong ring buffer, mặc định 5000 |

## Out of scope

- Replay một session từ event và audit log có giao diện: roadmap `audit-replay`.
- Retention / dọn bảng `events`: roadmap `retention` (TD-1).
- Đếm R/W tổng hợp theo kho dữ liệu cho web: tính từ bảng `events`, làm ở web hoặc feature sau.

## Open questions

| ID | Câu hỏi | Owner | Liên quan |
|---|---|---|---|
