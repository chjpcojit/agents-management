# Tasks — agent-registry

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
| Open | A-1, TD-1 |
| Knowledge | — |

Verification method: A `make lint`, B `make test` + `make test-integration`, C `make run` + curl `/agents`.

## Phase 1 — Domain

- [ ] [be][full] `domain.Agent` với validate theo bảng Field và state machine — verify: (B) `TestNewAgent_Valid`, `TestNewAgent_InvalidFields`, `TestPauseResume`, `TestInvalidStateTransitions`
- [ ] [be][full] `BasePolicy`, `EffectivePolicy`, `ParseToolRule` — verify: (B) `TestEffectivePolicyKeepsBase`

## Phase 2 — Persistence

- [ ] [be][full] Migration `agents` + FK `approvals.agent_id` — verify: (B) migrate up trên DB sạch (integration)
- [ ] [be][full] `AgentRepository` Postgres, `Update` atomic — verify: (B) `TestCreateSlugTaken`, contract test repository (integration)

## Phase 3 — Use case

- [ ] [be][full] `agent.Service` Create / Patch / Pause / Resume với event trong transaction — verify: (B) `TestCreateCanMessageValidation`, `TestPatchVersionConflict`, `TestPatchSlugImmutable`, `TestPatchEmitsChangedFields`
- [ ] [be][full] `Archive` với `RuntimeStatus.IsBusy`, gỡ `can_message`, huỷ mailbox — verify: (B) `TestArchiveBusy`, `TestArchiveCancelsQueuedAndUnlinks` — `mailbox.CancelQueued` thuộc agent-mailbox, dùng fake tới khi có

## Phase 4 — Delivery

- [ ] [be][full] `api/openapi.yaml` cho `/agents…` — verify: (A) lint OpenAPI
- [ ] [be][full] Handlers + error mapping — verify: (B) `TestListOmitsPrompt`, `TestGetIncludesEffectivePolicy`; (C) curl tạo, sửa, pause, archive

## Deferred

- Spawn theo cấu hình (R-9): [agent-supervisor](../agent-supervisor/tasks.md).
- Nội dung skill: roadmap `skill-management`.
