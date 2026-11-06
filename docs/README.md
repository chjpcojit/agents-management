# Docs

Tài liệu kỹ thuật của backend orchestrator. Quy tắc kiến trúc ở [`../CLAUDE.md`](../CLAUDE.md), cấu hình skill spec ở [`spec.config.yaml`](spec.config.yaml).

## Feature specs

MVP chạy local trên máy owner (Đức, 2026-11-05). Thứ tự implement theo cột #; 1 và 2 làm song song được.

| # | Feature | Mô tả | Phụ thuộc | Status |
|---|---|---|---|---|
| 0 | [agent-runtime](specs/features/agent-runtime/) | Chạy và điều khiển một process `claude -p` stream-json (spike 0) | — | In progress (spike xong) |
| 0 | [approval-inbox](specs/features/approval-inbox/) | Inbox duyệt tập trung: mở yêu cầu, quyết định, default deny khi hết hạn | storage | In progress |
| 1 | [event-schema](specs/features/event-schema/) | Envelope event v0, danh mục type, `resource` của tool call, kênh SSE `/events` | agent-runtime | Spec only |
| 2 | [storage](specs/features/storage/) | Postgres local qua docker compose, pgx + goose, transaction, repository atomic | — | Spec only |
| 3 | [agent-registry](specs/features/agent-registry/) | Tạo, sửa, tạm dừng, lưu trữ agent và cấu hình của nó | storage | Spec only |
| 4 | [data-access](specs/features/data-access/) | Nguồn dữ liệu ngoài (`dir`, `mcp`, `web`), grant theo agent, secret qua GCP Secret Manager | agent-registry | Spec only |
| 5 | [agent-supervisor](specs/features/agent-supervisor/) | Port `AgentRuntime`, một item một turn, session theo scope, vòng đời process | agent-runtime, agent-registry, data-access, event-schema | Spec only |
| 6 | [workspace](specs/features/workspace/) | Clone riêng, worktree + branch `YYYYMMDD-<agent>-<slug>` theo task | agent-registry | Spec only |
| 7 | [agent-mailbox](specs/features/agent-mailbox/) | Hàng đợi theo agent, tuần tự, `ask_agent`, reply đúng người hỏi, chống vòng lặp | agent-supervisor, storage, event-schema | Spec only |
| 8 | [task-dispatch](specs/features/task-dispatch/) | Giao việc, tin nhắn trong task, ngắt, huỷ, stream | agent-mailbox, workspace | Spec only |
| 9 | [concurrency-quota](specs/features/concurrency-quota/) | Scheduler: giới hạn turn đồng thời, chọn công bằng, dừng theo quota Max | agent-supervisor, agent-mailbox | Spec only |

Câu hỏi mở chặn một phần implement: data-access Q-1, Q-2; agent-supervisor Q-1; concurrency-quota Q-1. Cả bốn được trả lời bằng case spike T16…T19 (agent-runtime tasks).

## Roadmap sau MVP

| Phase | Feature | Nội dung |
|---|---|---|
| 2 | `office-bus` | Kênh chung, broadcast, nhóm được phép nói chuyện, token budget; xây trên agent-mailbox |
| 3 | `skill-management` | Xem, sửa, bật tắt skill global và skill trong worktree theo agent |
| 3 | `task-board` | Board theo assignee / status, log, task cha con |
| 4 | `push-notification` | Web push, Telegram cho approval và task xong (approval-inbox#TD-4) |
| 4 | `audit-replay` | Audit log ai duyệt gì, replay session từ event và transcript |
| 4 | `retention` | Dọn bảng `events`, transcript, worktree cũ (event-schema#TD-1) |
| — | `sandbox` | Cách ly OS cho `Bash` của agent (data-access#TD-1) |
| — | Deploy GCP | Ngoài GĐ1; secret đã ở GCP Secret Manager nên không đổi thiết kế secret |

## ADR

| ADR | Quyết định |
|---|---|
| [0001](adr/0001-workspace-worktree.md) | Workspace root, clone riêng, mỗi task một worktree có tiền tố ngày, push/PR qua inbox |
| [0002](adr/0002-postgres-local.md) | Postgres local qua docker compose, `pgx/v5` + `goose` |
| [0003](adr/0003-agent-messaging-async.md) | Agent hỏi nhau bất đồng bộ, reply lấy từ `result` cuối turn |
| [0004](adr/0004-secrets-gcp-secret-manager.md) | Secret của nguồn dữ liệu ngoài lưu ở GCP Secret Manager |
