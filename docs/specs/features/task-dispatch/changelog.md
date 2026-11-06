# Changelog — task-dispatch

## v1.0 — 2026-11-05

Initial, Prospective. Tạo task (worktree + item mailbox), tin nhắn trong task, ngắt, huỷ, đóng, trạng thái theo turn.

- requirements.md: R-1…R-13, SEC-1, SEC-2
- design.md: flow tạo, interrupt, cancel, projector, state machine, bảng `tasks`
- gherkin/task.feature: scenario cho R-1…R-13
- tasks.md: Phase 1–3
- assumptions.md, tech-debt.md: không có
- Upstream: none
- Reason: tách feature MVP
- Follow-ups outside this folder: web màn hình task (`MarkInterrupted` đã có ở agent-mailbox R-21)
