# Docs

Tài liệu kỹ thuật của backend orchestrator. Quy tắc kiến trúc ở [`../CLAUDE.md`](../CLAUDE.md), cấu hình skill spec ở [`spec.config.yaml`](spec.config.yaml).

## Feature specs

| Feature | Mô tả | Status |
|---|---|---|
| [approval-inbox](specs/features/approval-inbox/) | Inbox duyệt tập trung: mở yêu cầu, quyết định, default deny khi hết hạn | In progress |
| [agent-runtime](specs/features/agent-runtime/) | Chạy và điều khiển mỗi agent bằng một process `claude -p` stream-json (spike 0) | Spec only |
