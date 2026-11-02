# Tasks — agent-runtime

## Status

| Item | Value |
|---|---|
| Overall | In progress (spike code xong, chờ chạy trên máy owner) |
| Spec | v1.2 |
| Profile | be |
| Design | design.md |
| A — static | 2026-11-01: `make lint` → 0 issues |
| B — tests | 2026-11-01: `go test -race -count=1 ./...` → pass (claudecli 9.5s, mcpbus, approval, domain, cmd/spike 32.8s với 15/15 case trên claude giả) |
| C — driven | 2026-11-01: `go run ./cmd/spike preflight` trong container cloud → pass (claude 2.1.316); không chạy T1–T15 thật vì container không có login Max của owner |
| D — owner check | 2026-11-01, Đức, darwin/arm64, claude 2.1.316, Max: `make spike-test` → PASS T1, T2, T4, T8, T13, T15; OBSERVED T3, T9, T11; FAIL T5, T6, T7, T12, T14 (Bash bị rule của owner duyệt thẳng, sửa ở v1.2), T10 (lỗi check của spike, sửa ở v1.2). Report: `docs/spike-report.md` |
| E — e2e | not written |
| Branch / commit | `fix/spike-approval-bypass` (từ `spike/report`) |
| Open | Q-2 (deny), Q-5, Q-7, A-3, A-5, A-6, TD-3…TD-6 |
| Knowledge | — |

Verification method: A `make lint`, B `make test`, D `make spike-test` trên máy owner, kết quả ghi `docs/spike-report.md` và NDJSON thô ở `docs/spike-samples/` (gitignored, xem lại trước khi commit). Container cloud không có login Max nên không chạy được D ở đó.

## Phase 1 — Port và decoder

- [ ] Port `AgentRuntime`, `RuntimeEvent` — verify: (A) `make lint` — để sau spike, khi Q-1…Q-6 có câu trả lời (TD-4)
- [x] Decoder stream-json, giữ dòng lạ thành `unknown` — verify: (B) `TestDecodeGolden` — v1.1: pass, 2026-11-01

## Phase 2 — Runner

- [x] Spawn với env allowlist, flags cố định — verify: (B) `TestAllowedEnv`, `TestBuildArgs` — v1.1: pass, 2026-11-01
- [x] State machine idle/running/stopped/failed với process giả — verify: (B) `TestRunner*` — v1.1: pass, 2026-11-01
- [ ] Interrupt (SIGINT), shutdown, resume — verify: (B) pass 2026-11-01, (D) spike T12, T13 chưa chạy
- [x] Gửi khi đang chạy: ghi luôn, đếm `pending` — verify: (B) `TestSendWhileRunningQueues` — v1.2: pass, 2026-11-01; (D) T3 thấy Claude xếp hàng, 2026-11-01

## Phase 3 — MCP server

- [ ] `approval_prompt` nối approval-inbox — verify: (B) `TestApprovalPrompt*` pass 2026-11-01; còn blocked by Q-2 (D: T5–T7)
- [x] Định danh agent theo URL + bearer token — verify: (B) `TestAuth*`, `TestTwoAgentsRoutedSeparately`, (D) T15 — v1.2: pass, 2026-11-01
- [x] Stub `send_message` — verify: (B) `TestSendMessageStub`, (D) T8 — v1.2: pass, 2026-11-01

## Phase 4 — Spike chạy thật

- [x] `cmd/spike` preflight / run / test + Makefile `spike-preflight`, `spike`, `spike-test` — verify: (B) `TestPreflight*`, `TestScenariosAgainstFake`, `TestInteractiveSession` — v1.1: pass, 2026-11-01
- [x] Chạy T1–T15 trên máy owner lần 1 — verify: (D) `docs/spike-report.md` — 2026-11-01
- [ ] Ép tool ghi đi qua inbox bằng `permissions.ask` (SEC-5) — verify: (B) `TestBuildArgs`, `TestCheckNotBypassedCatchesUserRule` pass 2026-11-01; (D) chạy lại `SPIKE_ONLY=T5,T6,T7,T10,T12,T14`
- [ ] Case mới cho Q-7 (SIGINT khi còn prompt xếp hàng)
- [ ] Trả lời Q-2 (deny), Q-5, thay golden testdata bằng mẫu thật — verify: changelog

## Deferred

- Office bus đầy đủ: feature office-bus.
- Workspace và worktree: feature workspace ([ADR-0001](../../../adr/0001-workspace-worktree.md)).
