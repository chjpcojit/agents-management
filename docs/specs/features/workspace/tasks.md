# Tasks — workspace

## Status

| Item | Value |
|---|---|
| Overall | Spec only |
| Spec | v1.0 |
| Profile | be |
| Design | design.md |
| A — static | not run |
| B — tests | not run |
| C — driven | not yet |
| D — owner check | not yet |
| E — e2e | not written |
| Branch / commit | — |
| Open | A-1 |
| Knowledge | — |

Verification method: A `make lint` + grep không có lệnh xoá git, B `make test`, C `make run` + curl với repo công khai, D repo private qua SSH trên máy owner.

## Phase 1 — Domain và config

- [ ] [be][full] `WorktreeName`, `NextFreeName`, `ValidateRepoURL` — verify: (B) `TestWorktreeName`, `TestNextFreeName`, `TestValidateRepoURL`
- [ ] [be][full] `ORCH_WORKSPACE_ROOT` — verify: (B) `TestWorkspaceRootValidation`

## Phase 2 — Adapter và persistence

- [ ] [be][full] `gitcli` (argv, env cố định, timeout) — verify: (B) `TestGitEnv`; (C) clone repo công khai
- [ ] [be][full] Migration `projects`, `worktrees` + repository — verify: (B) contract test (integration)

## Phase 3 — Use case và delivery

- [ ] [be][full] `CreateProject` nền, `Retry`, project `cloning` dở chuyển `failed` lúc khởi động — verify: (B) `TestCreateProjectClones`, `TestCloneFailureRecorded`, `TestRetryProject`
- [ ] [be][full] `CreateWorktree` khoá theo project — verify: (B) `TestFetchFailureStillCreates`, `TestConcurrentWorktreesSameProject`, `TestWorktreeRecordedWithEvent`
- [ ] [be][full] Handlers + OpenAPI — verify: (B) `TestListWorktrees`; (C) curl
- [ ] [be][full] Repo private qua SSH agent — verify: (D) máy owner

## Deferred

- Dọn worktree: roadmap `retention`.
