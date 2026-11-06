# Tasks — concurrency-quota

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
| Open | Q-1, A-1, A-2, A-3, TD-1 |
| Knowledge | — |

Verification method: A `make lint`, B `make test`, D 3 agent thật với limit 2 trên máy owner.

## Phase 1 — Domain

- [ ] [be][full] `PickNext` — verify: (B) `TestPickNextOldestHead`
- [ ] [be][full] `ParseUsage`, `ShouldPause` với mẫu text spike T11 — verify: (B) `TestParseUsage`

## Phase 2 — Scheduler

- [ ] [be][full] Vòng lặp, slot, kick, Complete/Interrupt — verify: (B) `TestNeverExceedsLimit`, `TestKickTriggers`, `TestCompleteOrInterruptBySupervisorResult`, `TestFailedAgentSkipped`
- [ ] [be][full] Pause tay, pause quota, shutdown — verify: (B) `TestQuotaPauseAndAutoResume`, `TestPauseLetsRunningFinish`, `TestManualPauseNotClearedByQuota`, `TestShutdownStopsClaimingFirst`
- [ ] [be][full] `UsageProbe` + poller — verify: (B) với claude giả; (D) máy owner
- [ ] [be][full] Rate limit backoff — blocked by Q-1 (thêm case spike T18)

## Phase 3 — Delivery

- [ ] [be][full] `GET /scheduler`, pause, resume + OpenAPI — verify: (B) `TestSchedulerEndpoint`
- [ ] [be][full] Chạy thật 3 agent limit 2 — verify: (D)

## Deferred

- Ngân sách token theo agent: roadmap `office-bus`.
