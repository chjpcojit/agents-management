# Tasks — data-access

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
| Open | Q-1, Q-2, A-1, A-2, A-3, TD-1, TD-2 |
| Knowledge | — |

Verification method: A `make lint`, B `make test` + `make test-integration`, D trên máy owner (login Max, ADC GCP).

## Phase 1 — Domain

- [ ] [be][full] `Source` theo kind, đường dẫn cấm — verify: (B) `TestNewSource_ByKind`, `TestDirSourceDeniedPaths`
- [ ] [be][full] `Grant`, `BuildAccessPlan` theo bảng ánh xạ (chưa đưa `web`/`dir` vào plan tới khi có Q-1, Q-2) — verify: (B) `TestBuildAccessPlan`

## Phase 2 — Persistence và use case

- [ ] [be][full] Migration `data_sources`, `agent_grants` + repository — verify: (B) contract test (integration)
- [ ] [be][full] `access.Service` CRUD source, Put/Delete grant, thu hồi gọi `AgentStopper` — verify: (B) `TestUpdateSourceRevokesAgents`, `TestPutGrantAppliesNextSpawn`, `TestRevokeStopsAgent`
- [ ] [be][full] Handlers + `api/openapi.yaml` — verify: (B) `TestGetAccessHasNoSecretValues`; (C) curl

## Phase 3 — Secret và launch

- [ ] [be][full] `LaunchTokens` memory, redeem endpoint loopback — verify: (B) `TestLaunchTokenSingleUse`, `TestLaunchTokenExpiry`, `TestRedeemFailClosed`
- [ ] [be][full] `SecretResolver` GCP Secret Manager (ADC) — verify: (D) đọc một secret thử trên máy owner
- [ ] [be][full] Subcommand `mcp-launch` — verify: (B) `TestMcpLaunchExecsWithEnv`
- [ ] [be][full] Thêm case spike T16 (`WebFetch` domain allow vs ask) và T17 (rule path cho `--add-dir`) — verify: (D) `make spike-test SPIKE_ONLY=T16,T17` trên máy owner
- [ ] [be][full] Bật `web` grant trong plan — blocked by Q-1
- [ ] [be][full] Bật `dir` grant trong plan — blocked by Q-2

## Deferred

- Áp `AccessPlan` vào flags: [agent-supervisor](../agent-supervisor/tasks.md).
- Sandbox OS: TD-1. MCP HTTP có secret: TD-2.
