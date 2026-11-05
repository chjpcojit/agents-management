# Requirements — agent-registry

Scope: be (orchestrator)
Source: yêu cầu của Đức "tạo ra được các Agent, cấu hình cho nó các cài đặt cần thiết" (2026-11-02); danh sách feature MVP đã duyệt (2026-11-05)
Code: chưa có
Mode: Prospective

Hồ sơ của agent: tạo, xem, sửa, tạm dừng, lưu trữ agent và giữ cấu hình của nó. Feature này chỉ lưu; chạy process là việc của [agent-supervisor](../agent-supervisor/requirements.md), quyền vào dữ liệu ngoài là của [data-access](../data-access/requirements.md).

## Glossary

| Term | Nghĩa |
|---|---|
| Slug | Tên máy của agent, bất biến, dùng trong tên worktree và trong `ask_agent` |
| Config version | Số tăng mỗi lần cấu hình đổi; process đang chạy ghi lại version nó được spawn |
| Base policy | Rule quyền orchestrator luôn áp, config agent không gỡ được: `AskTools` (agent-runtime SEC-5), deny `.claude/**` (agent-runtime SEC-2) |

## User stories

- US-008: > As a **user**, I want **tạo agent với tên, model, prompt riêng, tool được dùng và agent nào được hỏi nó**, so that **mỗi agent có vai trò rõ ràng**.
- US-009: > As a **user**, I want **tạm dừng một agent mà không mất việc đang xếp hàng của nó**, so that **tôi kiểm soát được agent đang làm sai**.

## Acceptance criteria

### Tạo

- R-1: WHEN client gọi `POST /agents` với cấu hình hợp lệ, THE SYSTEM SHALL lưu agent với `status = active`, `config_version = 1`, ghi event `agent.created` trong cùng transaction, và trả 201 với agent. (platform)
- R-2: THE SYSTEM SHALL validate cấu hình theo bảng field ở [design](design.md#field) và trả 400 `invalid_argument` liệt kê mọi field lỗi cùng lúc, không lưu gì. (platform)
- R-3: IF `slug` đã được dùng bởi agent khác (kể cả agent `archived`), THEN THE SYSTEM SHALL trả 409 `slug_taken`. (platform)
- R-4: IF `model` không nằm trong `ORCH_ALLOWED_MODELS`, THEN THE SYSTEM SHALL trả 400 `invalid_argument` với field `model`. (platform)
- R-5: IF `can_message` chứa slug không tồn tại, slug của agent `archived`, hoặc chính slug của agent, THEN THE SYSTEM SHALL trả 400 `invalid_argument` với field `can_message`. (platform)

### Đọc

- R-6: WHEN client gọi `GET /agents`, THE SYSTEM SHALL trả 200 danh sách agent không `archived` (thêm `?include_archived=true` để lấy cả archived), mỗi phần tử không kèm `system_prompt`. (platform)
- R-7: WHEN client gọi `GET /agents/{id}`, THE SYSTEM SHALL trả 200 với cấu hình đầy đủ, kể cả `system_prompt` và `effective_policy` (allow / ask / deny sau khi gộp base policy); id không tồn tại trả 404. (platform)

### Sửa

- R-8: WHEN client gọi `PATCH /agents/{id}` với `config_version` hiện tại và các field cần đổi, THE SYSTEM SHALL áp thay đổi trong một `Update` atomic, tăng `config_version`, ghi event `agent.config_changed` chỉ chứa tên các field đã đổi, và trả 200, then:
  - on failure (`config_version` không khớp): 409 `version_conflict`, không đổi gì;
  - on failure (field không hợp lệ): 400 như R-2;
  - on failure (đổi `slug`): 400, `slug` bất biến. (platform)
- R-9: THE SYSTEM SHALL áp cấu hình mới ở lần spawn process tiếp theo; turn đang chạy giữ cấu hình cũ. (platform)

### Lọc tool và base policy

- R-10: THE SYSTEM SHALL tính `effective_policy` = base policy ∪ `disallowed_tools` (deny) ∪ (`allowed_tools` trừ tool thuộc ask của base policy); một tool có trong cả `allowed_tools` và base ask vẫn là ask. (platform)
- R-11: IF `allowed_tools` hoặc `disallowed_tools` có chuỗi không đúng cú pháp rule của Claude Code (`Tên` hoặc `Tên(pattern)`), THEN THE SYSTEM SHALL trả 400 với field đó. (platform)

### Trạng thái

- R-12: WHEN client gọi `POST /agents/{id}/pause` cho agent `active`, THE SYSTEM SHALL chuyển sang `paused` và ghi event `agent.status_changed`; agent `paused` vẫn nhận item vào mailbox nhưng không được giao item mới, turn đang chạy chạy nốt. (platform)
- R-13: WHEN client gọi `POST /agents/{id}/resume` cho agent `paused`, THE SYSTEM SHALL chuyển về `active` và ghi event. (platform)
- R-14: WHEN client gọi `DELETE /agents/{id}`, THE SYSTEM SHALL chuyển agent sang `archived`, ghi event, và không xoá bản ghi hay worktree, then:
  - on failure (agent đang có turn chạy): 409 `agent_busy`;
  - on success: các item còn `queued` trong mailbox bị huỷ theo [agent-mailbox](../agent-mailbox/requirements.md), agent bị gỡ khỏi `can_message` của mọi agent khác. (platform)
- R-15: IF thao tác trạng thái không hợp lệ với trạng thái hiện tại (pause agent đã `paused`, resume agent `active`, mọi thao tác trên agent `archived` trừ GET), THEN THE SYSTEM SHALL trả 409 `invalid_state`. (platform)

## Security

- SEC-1: THE SYSTEM SHALL không log `system_prompt`; log và event chỉ có id, slug, tên field đổi.
- SEC-2: THE SYSTEM SHALL không cho cấu hình agent gỡ base policy (R-10).
- SEC-3: Mọi endpoint ghi chỉ nhận từ client đã xác thực; tới khi có xác thực web, HTTP bind loopback (approval-inbox#TD-3).

## Contract surface

| Kind | Name | Owner | Notes |
|---|---|---|---|
| HTTP | `POST /agents`, `GET /agents`, `GET /agents/{id}`, `PATCH /agents/{id}`, `POST /agents/{id}/pause`, `POST /agents/{id}/resume`, `DELETE /agents/{id}` | this feature | shape trong `api/openapi.yaml` |
| Event | `agent.created`, `agent.config_changed`, `agent.status_changed` | this feature | envelope [event-schema](../event-schema/requirements.md) |
| Env var | `ORCH_ALLOWED_MODELS` | this feature | danh sách phân cách dấu phẩy, mặc định `opus,sonnet,haiku` |
| Table | `agents` | this feature | migration qua [storage](../storage/requirements.md) |
| Go port | `port.AgentRepository` | this feature | `Update` atomic |

## Out of scope

- Chạy, dừng, restart process: [agent-supervisor](../agent-supervisor/requirements.md).
- Nguồn dữ liệu và grant: [data-access](../data-access/requirements.md).
- Xem, sửa nội dung skill: roadmap `skill-management`; MVP chỉ lưu danh sách tên skill.
- Xoá cứng agent: không làm (dữ liệu audit).

## Open questions

| ID | Câu hỏi | Owner | Liên quan |
|---|---|---|---|
