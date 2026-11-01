# ADR-0001 — Workspace, clone riêng và worktree theo task

Status: Accepted · Date: 2026-11-01 · Decider: Đức

## Context

Owner muốn giao 2 agent làm 2 việc trên cùng một dự án, mỗi agent ra một PR. Hai agent sửa chung một thư mục sẽ giẫm lên nhau (file, index git, branch đang checkout). Orchestrator cũng không nên đụng vào bản clone owner đang tự code.

## Decision

1. Owner cấu hình **workspace root** một lần (ví dụ `~/agents-workspace`).
2. Orchestrator **clone riêng** repo của dự án vào workspace root, không dùng bản clone owner đang làm việc.
3. Mỗi task có **một git worktree và một branch riêng** tạo từ bản clone đó; agent chạy với `cwd` là worktree.
4. Tên worktree và branch có **tiền tố ngày** `YYYYMMDD-<agent>-<slug>`, ví dụ `20261101-agent-a-login`.
5. **MVP không tự cleanup.** Owner xem ngày trong tên và tự xóa worktree cũ bằng tay.
6. **`git push` và tạo PR phải qua approval inbox**, không bao giờ tự động; `ClassifyRisk` đã xếp `Bash` (gồm `git push`, `gh pr create`) là high.

## Consequences

- Hai agent trên cùng dự án chạy song song an toàn; mỗi PR đi từ branch riêng.
- Ổ đĩa tăng theo số task cho tới khi owner xóa; chấp nhận ở MVP.
- Branch trùng tên trong cùng ngày phải được xử lý (thêm hậu tố): để lại cho feature workspace.
- Spec chi tiết: feature `workspace` (chưa viết). Liên quan: [agent-runtime](../specs/features/agent-runtime/requirements.md), [approval-inbox](../specs/features/approval-inbox/requirements.md).
