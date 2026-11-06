# Requirements — workspace

Scope: be (orchestrator), `git` trên máy owner
Source: ADR-0001 (workspace root, clone riêng, worktree có ngày, push/PR qua inbox); danh sách feature MVP đã duyệt (2026-11-05)
Code: chưa có
Mode: Prospective

Mỗi task có bàn làm việc riêng: orchestrator clone riêng repo của dự án vào workspace root, rồi tạo một git worktree và một branch tên `YYYYMMDD-<agent>-<slug>` cho mỗi task. Hai agent làm song song trên cùng dự án không giẫm nhau. MVP không tự dọn; owner xoá tay theo ngày trong tên.

## Glossary

| Term | Nghĩa |
|---|---|
| Workspace root | Thư mục owner cấu hình một lần, chứa mọi clone và worktree |
| Project | Một repo đã clone riêng vào `<root>/<project-slug>/repo` |
| Worktree | `git worktree` của một task tại `<root>/<project-slug>/worktrees/<name>` |

## User stories

- US-013: > As a **user**, I want **giao hai việc trên cùng một repo cho hai agent và nhận hai PR riêng**, so that **chúng làm song song mà không phá nhau**.
- US-014: > As a **user**, I want **tên thư mục và branch có ngày**, so that **tôi biết cái nào cũ để tự xoá**.

## Acceptance criteria

### Workspace root

- R-1: WHEN orchestrator khởi động, THE SYSTEM SHALL đọc `ORCH_WORKSPACE_ROOT` (mặc định `~/agents-workspace`), tạo thư mục với quyền `0700` nếu chưa có, và từ chối khởi động (exit 1) nếu đường dẫn nằm trong một git repo khác hoặc không ghi được. (platform)

### Project

- R-2: WHEN client gọi `POST /projects` với `slug` (`^[a-z][a-z0-9-]{1,31}$`, unique) và `repo_url`, THE SYSTEM SHALL lưu project ở `cloning`, trả 202, rồi clone nền vào `<root>/<slug>/repo`, then:
  - on success: phát hiện default branch từ `origin/HEAD`, project `ready`, event `project.status_changed`;
  - on failure (clone lỗi: auth, mạng, repo không tồn tại): project `failed` với dòng lỗi đầu tiên của git đã lọc, event. (platform)
- R-3: IF `repo_url` không phải `https://…` hoặc dạng SSH (`git@host:path` hay `ssh://…`), THEN THE SYSTEM SHALL trả 400; `file://`, `ext::` và mọi transport khác bị từ chối. (platform)
- R-4: WHEN client gọi `POST /projects/{id}/retry` cho project `failed`, THE SYSTEM SHALL xoá thư mục clone dở và clone lại như R-2. (platform)
- R-5: THE SYSTEM SHALL dùng credential git sẵn có của owner (SSH agent, credential helper); orchestrator không nhận, lưu hay truyền token git. (platform)

### Worktree theo task

- R-6: WHEN task-dispatch yêu cầu worktree cho (project, agent slug, task title), THE SYSTEM SHALL `git fetch origin` rồi tạo worktree và branch cùng tên `YYYYMMDD-<agent-slug>-<task-slug>` từ `origin/<default_branch>`, với ngày theo giờ địa phương của máy và `task-slug` lấy từ title (bỏ dấu, chữ thường, `[a-z0-9-]`, tối đa 30 ký tự, rỗng thì `task`). (platform)
- R-7: IF tên đã có (thư mục, branch local hoặc branch trên `origin`), THEN THE SYSTEM SHALL thêm hậu tố `-2`, `-3`, … tới tên đầu tiên còn trống, tối đa `-99`. (platform)
- R-8: IF `git fetch` lỗi, THEN THE SYSTEM SHALL vẫn tạo worktree từ `origin/<default_branch>` đã có local và ghi cảnh báo vào event `worktree.created` (`fetched = false`). (platform)
- R-9: THE SYSTEM SHALL chạy tuần tự mọi thao tác git ghi (`clone`, `fetch`, `worktree add`) trên cùng một project. (platform)
- R-10: WHEN worktree tạo xong, THE SYSTEM SHALL lưu (task, project, name, path, branch, base commit) và ghi event `worktree.created`. (platform)

### Không dọn, không push hộ

- R-11: THE SYSTEM SHALL không tự xoá worktree, branch hay clone ở MVP, kể cả khi task `done`, `cancelled` hay agent `archived` (ADR-0001). (platform)
- R-12: THE SYSTEM SHALL không có API push hay tạo PR; agent tự chạy `git push` / `gh pr create` qua `Bash` và mọi lệnh đó đi qua approval inbox (ADR-0001, agent-runtime SEC-5). (platform)
- R-13: WHEN client gọi `GET /projects` hoặc `GET /projects/{id}/worktrees`, THE SYSTEM SHALL trả danh sách kèm path, branch, task, ngày tạo, để owner biết cái nào cũ. (platform)

## Security

- SEC-1: THE SYSTEM SHALL gọi `git` bằng argv (`exec.Command`), không qua shell, với env `GIT_ALLOW_PROTOCOL=https:ssh` và `GIT_TERMINAL_PROMPT=0`.
- SEC-2: THE SYSTEM SHALL lọc URL có userinfo (`https://user:token@…`) khỏi log, event và DB: URL có userinfo bị từ chối ở R-3.
- SEC-3: Env của process git theo allowlist của agent-runtime R-2 cộng `SSH_AUTH_SOCK`, `GIT_*` do orchestrator đặt.

## Contract surface

| Kind | Name | Owner | Notes |
|---|---|---|---|
| HTTP | `POST/GET /projects`, `GET /projects/{id}`, `POST /projects/{id}/retry`, `GET /projects/{id}/worktrees` | this feature | |
| Go | `workspace.Service.CreateWorktree(project, agentSlug, title, taskID)` | this feature | gọi bởi [task-dispatch](../task-dispatch/requirements.md) |
| Event | `project.status_changed`, `worktree.created` | this feature | |
| Env var | `ORCH_WORKSPACE_ROOT` | this feature | mặc định `~/agents-workspace` |
| Table | `projects`, `worktrees` | this feature | |
| Go port | `port.Git` | this feature | `Clone`, `Fetch`, `DefaultBranch`, `BranchExists`, `AddWorktree`, `HeadCommit` |

## Out of scope

- Dọn worktree, branch: roadmap `retention`; owner xoá tay.
- Cài dependency trong worktree: agent tự làm qua `Bash` (cần duyệt).
- Tạo PR, theo dõi CI: agent tự làm, push/PR qua inbox.
- Xử lý conflict khi hai agent sửa cùng file: lộ ra lúc merge PR, owner xử lý.

## Open questions

| ID | Câu hỏi | Owner | Liên quan |
|---|---|---|---|
