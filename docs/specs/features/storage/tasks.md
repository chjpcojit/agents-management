# Tasks — storage

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
| Open | A-1, A-2, TD-1, TD-2 |
| Knowledge | — |

Verification method: A `make lint`, B `make test` + `make test-integration` (cần `make db-up`), C `make db-up && make run` + `curl localhost:8787/readyz`. Container cloud có thể không có Docker; khi đó B integration và C chạy trên máy owner.

## Phase 1 — Môi trường

- [ ] [be][full] `compose.yaml`, `.env.example`, `.gitignore` có `.env` — verify: (C) `make db-up` + `pg_isready`; (A) `git check-ignore .env`
- [ ] [be][full] Target `db-up`, `db-down`, `db-psql`, `test-integration` — verify: (C) chạy từng target

## Phase 2 — Kết nối và migration

- [ ] [be][full] Config `ORCH_DATABASE_URL`, `ORCH_DB_MAX_CONNS`, `ORCH_DB_CONNECT_TIMEOUT`, redact — verify: (B) `TestLoadDatabaseConfig`, `TestConfigRedactsDatabaseURL`
- [ ] [be][full] `postgres.Open` retry + `goose.Up` embed + từ chối schema mới hơn — verify: (B) `TestOpenTimeout`, `TestMigrateUpAndRefuseNewerSchema`
- [ ] [be][full] `/readyz` ping 2s — verify: (B) `TestReadyz`

## Phase 3 — Transaction và repository

- [ ] [be][full] Port `TxRunner` + implement với retry `40001`/`40P01` — verify: (B) `TestWithinTxCommitRollbackPanic`, `TestRetryOnSerializationFailure`
- [ ] [be][full] Migration `00001_approvals.sql` + `ApprovalRepository` Postgres — verify: (B) `TestApprovalRepositoryContract`, `TestConcurrentDecisionsOnlyOneWins` (integration, `-race`)
- [ ] [be][full] `ExpireAllPending` lúc khởi động + wiring `cmd/orchestrator` + thứ tự shutdown — verify: (B) `TestExpirePendingOnStart`; (C) SIGTERM, đọc log

## Deferred

- Backup: TD-1. Role DB riêng: TD-2. Retention: roadmap.
