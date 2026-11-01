# Docs

Tài liệu kỹ thuật của backend orchestrator. Quy tắc kiến trúc ở [`../CLAUDE.md`](../CLAUDE.md), cấu hình skill spec ở [`spec.config.yaml`](spec.config.yaml).

## Feature specs

| Feature | Mô tả | Status |
|---|---|---|
| [approval-inbox](specs/features/approval-inbox/) | Inbox duyệt tập trung: mở yêu cầu, quyết định, default deny khi hết hạn | In progress |
| [agent-runtime](specs/features/agent-runtime/) | Chạy và điều khiển mỗi agent bằng một process `claude -p` stream-json (spike 0) | In progress (chờ chạy spike trên máy owner) |

## ADR

| ADR | Quyết định |
|---|---|
| [0001](adr/0001-workspace-worktree.md) | Workspace root, clone riêng, mỗi task một worktree có tiền tố ngày, push/PR qua inbox |
