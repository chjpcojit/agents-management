# Tasks — agent-runtime

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
| D — owner check | not yet (spike phải chạy trên máy owner với login Max) |
| E — e2e | not written |
| Branch / commit | — (dự kiến `spike/claude-p`) |
| Open | Q-1…Q-5, A-1, A-2, TD-1, TD-2 |
| Knowledge | — |

Verification method: A `make lint`, B `make test`, C/D chạy `cmd/spike` trên máy owner theo bảng T1–T15 trong spike spec; kết quả ghi `docs/spike-report.md`. Container cloud không có login Claude nên không chạy được C/D ở đó.

## Phase 1 — Port và decoder

- [ ] Port `AgentRuntime`, `RuntimeEvent` — verify: (A) `make lint`
- [ ] Decoder stream-json, giữ dòng lạ thành `unknown` — verify: (B) `TestDecoderGolden`

## Phase 2 — Runner

- [ ] Spawn với env allowlist, flags cố định — verify: (B) `TestRunnerEnvAllowlist`
- [ ] State machine starting/idle/running/stopped/failed với process giả — verify: (B) `TestRunnerStateMachine`
- [ ] Interrupt (SIGINT), shutdown, resume — verify: (B) `TestRunnerStateMachine`, (D) spike T12, T13
- [ ] Gửi khi đang chạy — blocked by Q-1

## Phase 3 — MCP server

- [ ] `approval_prompt` nối approval-inbox — blocked by Q-2
- [ ] Định danh agent theo URL — blocked by Q-4
- [ ] Stub `send_message` / `read_inbox` — verify: (D) spike T8

## Phase 4 — Spike chạy thật

- [ ] `cmd/spike` + chạy T1–T15 trên máy owner — verify: (D) `docs/spike-report.md`
- [ ] Trả lời Q-1…Q-5, cập nhật spec lên v1.1 — verify: changelog

## Deferred

- Office bus đầy đủ: feature office-bus.
- Worktree và quota: feature sau.
