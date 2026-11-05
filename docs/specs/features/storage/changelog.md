# Changelog — storage

## v1.0 — 2026-11-05

Initial, Prospective. Postgres local qua docker compose, pgx + goose, transaction, repository atomic, chuyển approval-inbox khỏi in-memory.

- requirements.md: R-1…R-14, SEC-1…SEC-4
- design.md: components, flow khởi động, transaction, update atomic, bảng `approvals`
- gherkin/database.feature: scenario cho R-1…R-14
- tasks.md: Phase 1–3
- assumptions.md: A-1, A-2; tech-debt.md: TD-1, TD-2
- Resolved: dùng Postgres local thay SQLite; driver `pgx/v5`, migration `goose` (Đức, 2026-11-02 và 2026-11-05) → ADR-0002
- Upstream: none
- Reason: tách feature MVP
- Follow-ups outside this folder: approval-inbox#TD-1 resolved bởi feature này (approval-inbox v1.4); README dự án thêm hướng dẫn `make db-up`
