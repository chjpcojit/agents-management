# Tasks — agent-supervisor

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

Verification method: A `make lint`, B `make test` (claude giả như spike), D trên máy owner với login Max.

## Phase 1 — Domain và port

- [ ] [be][full] `RuntimeState` + chuyển trạng thái — verify: (B) test bảng chuyển trạng thái
- [ ] [be][full] `BuildLaunchSpec` theo bảng LaunchSpec, file tạm cho prompt/settings/mcp-config — verify: (B) `TestBuildLaunchSpec`, `TestLaunchSpecNoPromptInArgs`
- [ ] [be][full] Port `AgentRuntime`, `RuntimeProcess`, `RuntimeEvent`, `SessionRepository`, `AgentStopper`, `RuntimeStatus` — verify: (A) `make lint`

## Phase 2 — Adapter

- [ ] [be][full] Bọc `claudecli.Runner` thành `AgentRuntime` — verify: (B) test runner hiện có chạy lại + `TestStopKillsWithinDeadline`
- [ ] [be][full] Migration `agent_sessions` + repository — verify: (B) contract test (integration)
- [ ] [be][full] `mcpbus` cấp / thu hồi bearer theo spawn — verify: (B) `TestBearerRotatedPerSpawn`

## Phase 3 — Use case

- [ ] [be][full] `RunTurn` với dùng lại / respawn theo scope và version — verify: (B) `TestRunTurnSpawnsAndReturnsResult`, `TestRunTurnSerial`, `TestOneProcessPerAgent`, `TestReuseIdleProcess`, `TestRespawnOnScopeOrVersionChange`, `TestSessionSavedAndResumed`
- [ ] [be][full] Mapper event + resources — verify: (B) `TestEventsMappedWithIDs`
- [ ] [be][full] Lỗi: exit, im lặng, auth, timeout; idle reaper; pause/archive; start offline — verify: (B) `TestExitDuringTurn`, `TestSilenceMarksFailed`, `TestAuthFailedMarksFailed`, `TestTurnTimeout`, `TestIdleReaper`, `TestPauseLetsTurnFinish`, `TestArchiveClosesIdle`, `TestStartAllOffline`, `TestRuntimeChangedEvents`
- [ ] [be][full] Dùng `--max-turns` — blocked by Q-1 (tạm đếm `num_turns`, TD-1)

## Phase 4 — Delivery

- [ ] [be][full] `GET /agents/{id}/runtime`, `POST …/reset` + OpenAPI — verify: (B) `TestRuntimeEndpoint`, `TestResetEndpoint`
- [ ] [be][full] Chạy thật 2 agent trên máy owner — verify: (D) kịch bản trong test plan

## Deferred

- Scheduler gọi `RunTurn`: [concurrency-quota](../concurrency-quota/tasks.md).
