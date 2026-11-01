# Tasks — agent-runtime

## Status

| Item | Value |
|---|---|
| Overall | In progress (spike code xong, chờ chạy trên máy owner) |
| Spec | v1.1 |
| Profile | be |
| Design | design.md |
| A — static | 2026-11-01: `make lint` → 0 issues |
| B — tests | 2026-11-01: `go test -race -count=1 ./...` → pass (claudecli 9.5s, mcpbus, approval, domain, cmd/spike 32.8s với 15/15 case trên claude giả) |
| C — driven | 2026-11-01: `go run ./cmd/spike preflight` trong container cloud → pass (claude 2.1.316); không chạy T1–T15 thật vì container không có login Max của owner |
| D — owner check | not yet: `make spike-preflight` rồi `make spike-test` trên máy Đức |
| E — e2e | not written |
| Branch / commit | `feat/agent-runtime-spike` |
| Open | Q-1…Q-6, A-1…A-4, TD-3…TD-5 |
| Knowledge | — |

Verification method: A `make lint`, B `make test`, D `make spike-test` trên máy owner, kết quả ghi `docs/spike-report.md` và NDJSON thô ở `docs/spike-samples/` (gitignored, xem lại trước khi commit). Container cloud không có login Max nên không chạy được D ở đó.

## Phase 1 — Port và decoder

- [ ] Port `AgentRuntime`, `RuntimeEvent` — verify: (A) `make lint` — để sau spike, khi Q-1…Q-6 có câu trả lời (TD-4)
- [x] Decoder stream-json, giữ dòng lạ thành `unknown` — verify: (B) `TestDecodeGolden` — v1.1: pass, 2026-11-01

## Phase 2 — Runner

- [x] Spawn với env allowlist, flags cố định — verify: (B) `TestAllowedEnv`, `TestBuildArgs` — v1.1: pass, 2026-11-01
- [x] State machine idle/running/stopped/failed với process giả — verify: (B) `TestRunner*` — v1.1: pass, 2026-11-01
- [ ] Interrupt (SIGINT), shutdown, resume — verify: (B) pass 2026-11-01, (D) spike T12, T13 chưa chạy
- [ ] Gửi khi đang chạy — blocked by Q-1

## Phase 3 — MCP server

- [ ] `approval_prompt` nối approval-inbox — verify: (B) `TestApprovalPrompt*` pass 2026-11-01; còn blocked by Q-2 (D: T5–T7)
- [ ] Định danh agent theo URL + bearer token — verify: (B) `TestAuth*`, `TestTwoAgentsRoutedSeparately` pass 2026-11-01; còn blocked by Q-4 (D: T15)
- [ ] Stub `send_message` — verify: (B) `TestSendMessageStub` pass 2026-11-01, (D) spike T8 chưa chạy

## Phase 4 — Spike chạy thật

- [x] `cmd/spike` preflight / run / test + Makefile `spike-preflight`, `spike`, `spike-test` — verify: (B) `TestPreflight*`, `TestScenariosAgainstFake`, `TestInteractiveSession` — v1.1: pass, 2026-11-01
- [ ] Chạy T1–T15 trên máy owner — verify: (D) `docs/spike-report.md`
- [ ] Trả lời Q-1…Q-6, cập nhật spec lên v1.2, thay golden testdata bằng mẫu thật — verify: changelog

## Deferred

- Office bus đầy đủ: feature office-bus.
- Workspace và worktree: feature workspace ([ADR-0001](../../../adr/0001-workspace-worktree.md)).
