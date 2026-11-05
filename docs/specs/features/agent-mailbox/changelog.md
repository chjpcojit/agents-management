# Changelog — agent-mailbox

## v1.0 — 2026-11-05

Initial, Prospective. Mailbox bền theo agent, giao tuần tự một item một turn, `ask_agent` bất đồng bộ, reply tự động, idempotency, chống vòng lặp.

- requirements.md: R-1…R-21, SEC-1…SEC-3
- design.md: flow hỏi và giao/hoàn tất, state machine, prompt header, bảng `mailbox_items`
- gherkin/mailbox.feature: scenario cho R-1…R-21
- tasks.md: Phase 1–4
- assumptions.md: A-1; tech-debt.md: không có
- Resolved: agent hỏi bất đồng bộ, reply lấy từ `result` cuối turn (Đức, 2026-11-05) → ADR-0003
- Upstream: none
- Reason: yêu cầu của Đức (2026-11-02)
- Follow-ups outside this folder: agent-runtime#TD-3 resolved khi Phase 4 xong; web màn hình hội thoại giữa agent
