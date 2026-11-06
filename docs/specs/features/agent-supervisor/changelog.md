# Changelog — agent-supervisor

## v1.0 — 2026-11-05

Initial, Prospective. Port `AgentRuntime`, `RunTurn` một item một turn, scope và session, vòng đời process theo agent.

- requirements.md: R-1…R-18, SEC-1…SEC-3, Q-1
- design.md: LaunchSpec, chọn scope, flow RunTurn và Dừng, state machine, bảng `agent_sessions`
- gherkin/agent-supervisor.feature: scenario cho R-1…R-18
- tasks.md: Phase 1–4, một item blocked by Q-1
- assumptions.md: A-1…A-3; tech-debt.md: TD-1
- Upstream: agent-runtime v1.5
- Reason: tách feature MVP; resolve agent-runtime#TD-4
- Follow-ups outside this folder: agent-runtime ghi rằng đường scheduler không dùng R-8 (agent-runtime v1.6)
