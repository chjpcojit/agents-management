# Requirements — concurrency-quota

Scope: be (orchestrator), `claude -p "/usage"` trên máy owner
Source: design doc dự án ("quota Max chung (5h + tuần) → concurrency limit + queue", 2026-11-01); agent-runtime T11, agent-runtime#TD-6; danh sách feature MVP đã duyệt (2026-11-05)
Code: chưa có
Mode: Prospective

Mọi agent dùng chung quota gói Max của owner (cửa sổ 5 giờ và theo tuần). Scheduler là vòng lặp duy nhất quyết định **agent nào được chạy turn tiếp theo**: giới hạn số turn chạy đồng thời, chọn công bằng giữa các agent có việc, và ngừng giao việc mới khi quota gần hết. Mỗi lượt giao: claim item ([agent-mailbox](../agent-mailbox/requirements.md)) → chạy turn ([agent-supervisor](../agent-supervisor/requirements.md)) → báo kết quả về mailbox.

## Glossary

| Term | Nghĩa |
|---|---|
| Slot | Quyền chạy một turn; số slot = `ORCH_MAX_RUNNING` |
| Eligible agent | Agent `active`, runtime không `failed`, không đang chạy turn, có item `queued` |
| Quota snapshot | % đã dùng của phiên 5 giờ và của tuần, kèm thời điểm reset, đọc từ `/usage` |

## User stories

- US-019: > As a **user**, I want **giới hạn số agent chạy cùng lúc và để phần còn lại xếp hàng**, so that **không đốt hết quota 5 giờ trong vài phút**.
- US-020: > As a **user**, I want **orchestrator tự ngừng giao việc khi quota gần hết và báo tôi**, so that **agent không bị cắt giữa chừng vì hết quota**.

## Acceptance criteria

### Giới hạn và chọn agent

- R-1: THE SYSTEM SHALL không bao giờ có quá `ORCH_MAX_RUNNING` (mặc định 2, 1…8) turn đang chạy cùng lúc trên mọi agent. (platform)
- R-2: WHEN có slot trống và scheduler không bị dừng, THE SYSTEM SHALL chọn trong các eligible agent agent có item đầu hàng tạo sớm nhất (hoà thì theo `agent_id`), claim item đó và chạy turn. (platform)
- R-3: THE SYSTEM SHALL xét giao việc khi có item mới được đưa vào, khi một turn kết thúc, khi agent được resume hay reset, khi scheduler được resume, và ít nhất mỗi 5 giây. (platform)
- R-4: WHEN turn kết thúc, THE SYSTEM SHALL trả slot và báo mailbox `Complete` (có `TurnResult`) hoặc `Interrupt` (turn bị gián đoạn hay quá giờ); agent `failed` bị bỏ qua, item của nó giữ `queued`. (platform)

### Quota

- R-5: WHILE orchestrator chạy, THE SYSTEM SHALL mỗi `ORCH_QUOTA_POLL` (mặc định 5m) chạy một process ngắn `claude -p "/usage"` trong thư mục riêng của orchestrator, parse % phiên 5 giờ, % tuần và thời điểm reset, rồi ghi event `quota.updated`. (platform)
- R-6: WHEN snapshot có % phiên hoặc % tuần ≥ `ORCH_QUOTA_PAUSE_PCT` (mặc định 90), THE SYSTEM SHALL dừng giao turn mới với lý do `quota`, để turn đang chạy chạy nốt, và ghi event `scheduler.changed`; WHEN snapshot sau đó dưới ngưỡng, THE SYSTEM SHALL tự giao tiếp nếu không bị owner dừng tay. (platform)
- R-7: IF output `/usage` không parse được hoặc process lỗi, THEN THE SYSTEM SHALL ghi `quota.updated` với `status = unknown` kèm text gốc tối đa 500 ký tự và không dừng scheduler. (platform)
- R-8: WHEN một turn báo lỗi rate limit của gói (⚠️ Q-1), THE SYSTEM SHALL dừng giao turn mới với lý do `rate_limited` trong `ORCH_RATE_LIMIT_BACKOFF` (mặc định 5m) hoặc tới thời điểm reset nếu biết, rồi tự giao tiếp. (platform)

### Điều khiển tay và quan sát

- R-9: WHEN client gọi `POST /scheduler/pause` hoặc `POST /scheduler/resume`, THE SYSTEM SHALL dừng hoặc tiếp tục giao turn mới, ghi event; dừng tay không bị quota tự gỡ. (platform)
- R-10: WHEN client gọi `GET /scheduler`, THE SYSTEM SHALL trả 200 với `limit`, danh sách turn đang chạy (agent, item, bắt đầu lúc), số item `queued` theo agent, `paused` và `reason`, quota snapshot mới nhất. (platform)
- R-11: WHEN orchestrator shutdown, THE SYSTEM SHALL ngừng claim item mới trước tiên, rồi để agent-supervisor dừng các turn đang chạy (agent-runtime R-14). (platform)

## Security

- SEC-1: Process `/usage` chạy với cùng env allowlist như agent (agent-runtime R-2), không MCP, không tool, trong thư mục không chứa code.

## Contract surface

| Kind | Name | Owner | Notes |
|---|---|---|---|
| HTTP | `GET /scheduler`, `POST /scheduler/pause`, `POST /scheduler/resume` | this feature | |
| Event | `scheduler.changed`, `quota.updated` | this feature | |
| Env var | `ORCH_MAX_RUNNING`, `ORCH_QUOTA_POLL`, `ORCH_QUOTA_PAUSE_PCT`, `ORCH_RATE_LIMIT_BACKOFF` | this feature | 2; 5m; 90; 5m |
| CLI output | text `/usage` ("Current session: 45% used · resets …", "Current week (all models): 31% used · resets …") | Anthropic | không có schema, agent-runtime#TD-6 |
| Go | `mailbox.Service.ClaimNext/Complete/Interrupt`, `supervisor.Service.RunTurn` | features khác | |

## Out of scope

- Ưu tiên theo agent hay theo task: FIFO công bằng theo thời gian tạo item (agent-mailbox A-1).
- Đổi `ORCH_MAX_RUNNING` khi đang chạy: đổi env và khởi động lại.
- Ngân sách token theo agent: roadmap `office-bus`.

## Open questions

| ID | Câu hỏi | Owner | Liên quan |
|---|---|---|---|
| Q-1 | Khi hết quota gói Max, stream-json báo bằng gì (`system/api_retry` với `error` nào, `result` subtype nào, có thời điểm reset không)? | spike trên máy Đức | R-8, A-2 |
