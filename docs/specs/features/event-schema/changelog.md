# Changelog — event-schema

## v1.0 — 2026-11-05

Initial, Prospective. Envelope v0, danh mục type, resource cho tool call, kênh SSE chung.

- requirements.md: R-1…R-17, SEC-1, SEC-2
- design.md: components, flows ghi/phát và kết nối lại, danh mục type, bảng `events`
- gherkin/event-stream.feature: scenario cho R-1…R-17
- tasks.md: Phase 1–3
- assumptions.md: A-1, A-2; tech-debt.md: TD-1, TD-2
- Upstream: none (danh sách feature MVP đã duyệt, Đức 2026-11-05)
- Reason: tách feature MVP
- Follow-ups outside this folder: approval-inbox chuyển SSE sang kênh này (approval-inbox v1.4); web (`agents-website`) đổi client SSE sang envelope v0
