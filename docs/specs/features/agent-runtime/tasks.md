# Tasks — agent-runtime

## Status

| Item | Value |
|---|---|
| Overall | In progress (spike xong trên máy owner; còn port `AgentRuntime` và golden testdata thật) |
| Spec | v1.6 |
| Profile | be |
| Design | design.md |
| A — static | 2026-11-01: `make lint` → 0 issues |
| B — tests | 2026-11-01: `go test -race -count=1 ./...` → pass (claudecli 9.5s, mcpbus, approval, domain, cmd/spike 32.8s với 15/15 case trên claude giả) |
| C — driven | 2026-11-01: `go run ./cmd/spike preflight` trong container cloud → pass (claude 2.1.316); không chạy T1–T15 thật vì container không có login Max của owner |
| D — owner check | 2026-11-01, Đức, darwin/arm64, claude 2.1.316, Max: `make spike-test` → PASS T1, T2, T4, T8, T13, T15; OBSERVED T3, T9, T11; FAIL T5, T6, T7, T12, T14 (Bash bị rule của owner duyệt thẳng, sửa ở v1.2), T10 (lỗi check của spike, sửa ở v1.2). Report: `docs/spike-report.md`<br>2026-11-02, lần 2 (`SPIKE_ONLY=T3,T5,T6,T7,T10,T12,T14`): PASS T5, T6, T10, T12, T14; OBSERVED T3; FAIL T7 (Claude Code bỏ chờ duyệt sau 60s, sửa ở v1.3). T12 chưa thật sự cắt lệnh vì Claude chặn `sleep` đứng riêng<br>2026-11-02, lần 3 (`SPIKE_ONLY=T7,T12`): PASS T7 (chờ duyệt 2 phút, A-7); FAIL T12 (SIGINT lúc `ping` đang chạy làm `claude` thoát exit 0, không có `result`; đổi sang Stop + resume ở v1.4)<br>2026-11-02, lần 4 (`SPIKE_ONLY=T12,T14`): PASS T12 (dừng trong 608ms, exit 143, không có `t12_done.txt` sau 20s, resume nhớ từ khóa), PASS T14. Mọi case T1–T15 đã PASS hoặc OBSERVED |
| E — e2e | not written |
| Branch / commit | `main` @ 55009e8 |
| Open | A-5, TD-3…TD-7 |
| Knowledge | — |

Verification method: A `make lint`, B `make test`, D `make spike-test` trên máy owner, kết quả ghi `docs/spike-report.md` và NDJSON thô ở `docs/spike-samples/` (gitignored, xem lại trước khi commit). Container cloud không có login Max nên không chạy được D ở đó.

## Phase 1 — Port và decoder

- [ ] Port `AgentRuntime`, `RuntimeEvent` — làm ở [agent-supervisor](../agent-supervisor/tasks.md) Phase 1 (TD-4)
- [x] Decoder stream-json, giữ dòng lạ thành `unknown` — verify: (B) `TestDecodeGolden` — v1.1: pass, 2026-11-01

## Phase 2 — Runner

- [x] Spawn với env allowlist, flags cố định — verify: (B) `TestAllowedEnv`, `TestBuildArgs` — v1.1: pass, 2026-11-01
- [x] State machine idle/running/stopped/failed với process giả — verify: (B) `TestRunner*` — v1.1: pass, 2026-11-01
- [x] Stop (kill process group), shutdown, resume — verify: (B) `TestStopEndsRunningTurn`, `TestStopKillsChildProcesses`, `TestExitReapsLeftoverChildren`, (D) T12 lần 4, T13 — v1.5: pass, 2026-11-02
- [x] Gửi khi đang chạy: ghi luôn, đếm `pending` — verify: (B) `TestSendWhileRunningQueues` — v1.2: pass, 2026-11-01; (D) T3 thấy Claude xếp hàng, 2026-11-01

## Phase 3 — MCP server

- [x] `approval_prompt` nối approval-inbox — verify: (B) `TestApprovalPrompt*`, (D) T5, T6 lần 2 — v1.3: pass, 2026-11-02
- [x] Định danh agent theo URL + bearer token — verify: (B) `TestAuth*`, `TestTwoAgentsRoutedSeparately`, (D) T15 — v1.2: pass, 2026-11-01
- [x] Stub `send_message` — verify: (B) `TestSendMessageStub`, (D) T8 — v1.2: pass, 2026-11-01

## Phase 4 — Spike chạy thật

- [x] `cmd/spike` preflight / run / test + Makefile `spike-preflight`, `spike`, `spike-test` — verify: (B) `TestPreflight*`, `TestScenariosAgainstFake`, `TestInteractiveSession` — v1.1: pass, 2026-11-01
- [x] Chạy T1–T15 trên máy owner lần 1 — verify: (D) `docs/spike-report.md` — 2026-11-01
- [x] Ép tool ghi đi qua inbox bằng `permissions.ask` (SEC-5) — verify: (B) `TestBuildArgs`, `TestCheckNotBypassedCatchesUserRule`, (D) T5, T6 lần 2 — v1.3: pass, 2026-11-02
- [x] Nâng giới hạn chờ duyệt bằng `timeout` trong mcp-config (R-16) — verify: (B) `TestBuildArgs`, (D) T7 lần 3 — v1.4: pass, 2026-11-02
- [x] Approval bị bỏ khi Claude hủy lệnh gọi (R-17) — verify: (B) `TestAbandon`, `TestAbandonEndsPendingOnly`, `TestCancelledCallAbandonsApproval` — v1.3: pass, 2026-11-02
- [x] T12: Stop giữa lệnh `ping`, không còn process mồ côi, resume nhớ hội thoại — verify: (B) `TestScenariosAgainstFake`, (D) `SPIKE_ONLY=T12,T14 make spike-test` — v1.5: pass, 2026-11-02
- [ ] Thay golden testdata bằng mẫu thật — verify: (B) `TestDecodeGolden`

## Phase spike bổ sung

- [ ] Case T16: rule `WebFetch(domain:x)` allow có thắng `ask` chung không — verify: (D) `make spike-test SPIKE_ONLY=T16` (data-access Q-1)
- [ ] Case T17: rule path cho thư mục `--add-dir` ở `-p` — verify: (D) `SPIKE_ONLY=T17` (data-access Q-2)
- [ ] Case T18: tín hiệu rate limit / hết quota trong stream-json — verify: (D) `SPIKE_ONLY=T18` (concurrency-quota Q-1)
- [ ] Case T19: `--max-turns` với stdin giữ mở — verify: (D) `SPIKE_ONLY=T19` (agent-supervisor Q-1)

## Deferred

- Office bus đầy đủ: feature office-bus.
- Workspace và worktree: feature workspace ([ADR-0001](../../../adr/0001-workspace-worktree.md)).
