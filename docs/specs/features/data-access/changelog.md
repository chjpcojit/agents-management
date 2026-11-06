# Changelog — data-access

## v1.0 — 2026-11-05

Initial, Prospective. Nguồn dữ liệu ngoài (`dir`, `mcp`, `web`), grant theo agent, AccessPlan, secret qua GCP Secret Manager và wrapper `mcp-launch`.

- requirements.md: R-1…R-15, SEC-1…SEC-5, Q-1, Q-2
- design.md: bảng ánh xạ grant → AccessPlan, flow thu hồi, flow spawn MCP có secret
- gherkin/data-access.feature: scenario cho R-1…R-15
- tasks.md: Phase 1–3, hai item blocked by Q-1, Q-2
- assumptions.md: A-1…A-3; tech-debt.md: TD-1, TD-2
- Resolved: secret lưu ở GCP Secret Manager (Đức, 2026-11-05) → ADR-0004
- Upstream: none
- Reason: yêu cầu của Đức (2026-11-02)
- Follow-ups outside this folder: agent-runtime thêm case spike T16, T17; hướng dẫn setup ADC + IAM trong README; web màn hình nguồn dữ liệu và quyền
