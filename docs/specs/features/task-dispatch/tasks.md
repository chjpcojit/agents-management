# Tasks — task-dispatch

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
| Open | — |
| Knowledge | — |

Verification method: A `make lint`, B `make test` + `make test-integration`, C/D giao task thật trên máy owner và theo dõi `/events?task_id=`.

## Phase 1 — Domain và persistence

- [ ] [be][full] `domain.Task` + state machine — verify: (B) test bảng chuyển trạng thái
- [ ] [be][full] Migration `tasks` + repository — verify: (B) contract test (integration)

## Phase 2 — Use case

- [ ] [be][full] `Create` (idempotency, validate, worktree, tx với mailbox) — verify: (B) `TestCreateTaskAtomic`, `TestCreateTaskIdempotent`, `TestCreateTaskValidation`, `TestCreateTaskMailboxFullKeepsWorktree`
- [ ] [be][full] Projector theo event mailbox — verify: (B) `TestProjectorRunningIdleFailed`
- [ ] [be][full] `SendMessage`, `Interrupt` (cần `mailbox.MarkInterrupted`), `Cancel`, `Complete` — verify: (B) `TestSendMessage`, `TestInterruptResumesSameSession`, `TestCancelCancelsQueued`, `TestComplete`

## Phase 3 — Delivery

- [ ] [be][full] Handlers + OpenAPI — verify: (B) `TestTaskEndpoints`
- [ ] [be][full] Giao task thật, ngắt, nhắn tiếp — verify: (D) máy owner

## Deferred

- Board: roadmap `task-board`.
