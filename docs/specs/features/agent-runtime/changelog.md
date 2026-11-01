# Changelog — agent-runtime

## v1.1 — 2026-11-01

Implement spike: decoder, runner, MCP server, `cmd/spike` với T1–T15 tự động; spec cập nhật theo code.

- requirements.md: header Code; R-2 allowlist đầy đủ; R-3 timeout 90s đo từ lúc gửi prompt; R-8, R-10 ghi hành vi tạm; SEC-4 viết thành EARS (path + bearer token); contract surface thêm `send_message`, make targets; out of scope trỏ ADR-0001; thêm Q-6
- design.md: components theo code thật (`internal/delivery/mcpbus`, `protocol`, `cmd/spike`); step failure 16 MB, 90s, 401; state machine bỏ `starting`; test plan theo test đã có
- tasks.md: tick decoder, runner spawn/state machine, `cmd/spike` (lệnh đã chạy ghi ở Status); MCP items còn blocked bởi Q-2, Q-4
- gherkin/agent-session.feature: R-3 viết lại (lỗi auth lộ ra ở prompt đầu), thêm scenario R-3 im lặng 90s và SEC-4; vẫn `@wip` tới khi có kết quả D
- assumptions.md / tech-debt.md: thêm A-3, A-4, TD-3, TD-4, TD-5; TD-1, TD-2 resolved
- Upstream: unchanged
- Reason: Đức yêu cầu code spike trên cloud, PR về, anh chạy bằng lệnh make trên máy mình
- Follow-ups outside this folder: `docs/adr/0001-workspace-worktree.md` (quyết định workspace của Đức), approval-inbox v1.2 (Q-2 push deferred), `.golangci.yml` cho phép test của delivery dùng adapter thật

## v1.0 — 2026-11-01

v1.0 (initial), Prospective, migrate từ spike 0 spec và báo cáo kiểm chứng 5 điểm mở.

- requirements.md: R-1…R-15, SEC-1…SEC-4, Q-1…Q-5; test case T1–T15 của spike map vào R-n trong design test plan
- design.md: components, flow gửi prompt và duyệt quyền (steps + diagram), state machine, error mapping
- tasks.md: Phase 1–4, chưa tick
- gherkin/agent-session.feature: scenario cho R-1…R-15, toàn bộ `@wip`
- assumptions.md / tech-debt.md: A-1, A-2, TD-1, TD-2
- Upstream: docs Claude Code tại code.claude.com, đọc 2026-11-01
- Reason: Đức yêu cầu tạo tài liệu bằng skill spec
- Follow-ups outside this folder: none
