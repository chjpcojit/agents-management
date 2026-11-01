# Tasks — approval-inbox

## Status

| Item | Value |
|---|---|
| Overall | In progress |
| Spec | v1.2 |
| Profile | be |
| Design | design.md |
| A — static | 2026-11-01: `make lint` → 0 issues; depguard thử import sai domain → platform và usecase → adapter, đều bị báo |
| B — tests | 2026-11-01: `make test` (`go test -race -count=1 ./...`) → domain, approval, config, httpapi pass, 0 failure |
| C — driven | 2026-11-01: `make build`, chạy binary, `curl /healthz` → `{"status":"ok"}`, SIGINT → shutdown sạch. Chưa drive approval vì chưa có delivery |
| D — owner check | not yet |
| E — e2e | not written |
| Branch / commit | `chore/clean-architecture-skeleton` |
| Open | TD-1, TD-2, TD-3, TD-4 |
| Knowledge | — |

Verification method: A `make lint`, B `make test`, C `make run` + curl. Môi trường: container cloud, không có DB, không có login Claude.

## Phase 1 — Domain

- [x] `Approval` + `NewApproval` validate và copy input — verify: (B) `TestNewApproval_Valid`, `TestNewApproval_Invalid`, `TestNewApproval_CopiesInput` — v1.0: pass, 2026-11-01
- [x] State machine `Decide` / `Expire` — verify: (B) `TestDecide`, `TestDecide_Twice`, `TestExpire` — v1.0: pass, 2026-11-01

## Phase 2 — Use case

- [x] `Service.Request` / `Decide` / `ExpireDue` — verify: (B) `internal/usecase/approval/service_test.go` — v1.0: pass, 2026-11-01
- [x] Quyết định đồng thời chỉ một thắng — verify: (B) `TestConcurrentDecisionsOnlyOneWins` với `-race` — v1.0: pass, 2026-11-01
- [ ] Test lỗi một bản ghi khi quét không dừng vòng quét (R-11) — verify: (B) test với fake repository lỗi

## Phase 3 — Adapters và wiring

- [x] Repository in-memory với `Update` atomic — verify: (B) qua test use case — v1.0: pass, 2026-11-01
- [x] Sweeper 1s trong `cmd/orchestrator`, dừng theo context — verify: (C) chạy binary, SIGINT, log `stopped` — v1.0: pass, 2026-11-01
- [ ] Repository SQLite — blocked by TD-1 (chờ chọn driver)

## Phase 4 — Delivery

- [ ] `api/openapi.yaml` cho `/approvals`, `/approvals/{id}`, `/approvals/{id}/decision`, `/events`, problem body — verify: (A) lint OpenAPI
- [ ] Port `EventPublisher` + `eventbus/memory` ring buffer — verify: (B) `TestSSEPublishAndReplay`
- [ ] Service phát event sau khi lưu thành công — verify: (B) test use case với fake publisher
- [ ] Handler list + get + decide, map lỗi sang RFC 7807 — verify: (B) `TestListPendingApprovals`, `TestGetApproval`, `TestDecideHandlerStatusMapping`. Không bật ngoài loopback khi TD-3 còn mở
- [ ] SSE `/events` với heartbeat và `Last-Event-ID` — verify: (B) `TestSSEPublishAndReplay`, (C) curl `-N`

## Deferred

- Push notification cho risk `high`: hoãn theo quyết định của Đức (TD-4).
- MCP `approval_prompt` nối agent vào inbox: [agent-runtime](../agent-runtime/tasks.md).
