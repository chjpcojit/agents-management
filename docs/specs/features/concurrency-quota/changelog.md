# Changelog — concurrency-quota

## v1.0 — 2026-11-05

Initial, Prospective. Scheduler duy nhất, giới hạn turn đồng thời, chọn công bằng, dừng theo quota `/usage`, dừng tay.

- requirements.md: R-1…R-11, SEC-1, Q-1
- design.md: flow tick và quota poll, cờ dừng, config
- gherkin/scheduler.feature: scenario cho R-1…R-11 (R-8 `@wip`)
- tasks.md: Phase 1–3, một item blocked by Q-1
- assumptions.md: A-1…A-3; tech-debt.md: TD-1
- Upstream: agent-runtime v1.5 (T11)
- Reason: tách feature MVP
- Follow-ups outside this folder: agent-runtime thêm case spike T18 (rate limit)
