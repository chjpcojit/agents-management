# Changelog — agent-runtime

## v1.6 — 2026-11-05

Nối agent-runtime với các feature MVP mới và thêm 4 case spike cho câu hỏi mở của chúng.

- requirements.md: ghi chú dưới R-8 (đường mailbox không ghi chồng prompt); `send_message` chuyển sang agent-mailbox
- design.md: no change needed, checked
- tasks.md: port `AgentRuntime` chuyển sang agent-supervisor; thêm case spike T16…T19
- gherkin/agent-session.feature: no change needed, checked
- assumptions.md: no change needed, checked
- tech-debt.md: Fix của TD-3 (agent-mailbox), TD-4 (agent-supervisor)
- Upstream: unchanged
- Reason: tách feature MVP; ADR-0003
- Follow-ups outside this folder: none

## v1.5 — 2026-11-02

Lần chạy spike thứ 4 xác nhận cách hủy mới: dừng process group không để lại lệnh mồ côi và resume giữ hội thoại.

- requirements.md: no change needed, checked
- design.md: no change needed, checked
- tasks.md: Status D ghi lần 4; tick Stop/shutdown/resume và T12
- gherkin/agent-session.feature: no change needed, checked
- assumptions.md / tech-debt.md: no change needed, checked
- Upstream: claude 2.1.316
- Reason: report lần 4 `docs/spike-report.md` (Đức, 2026-11-02, commit 55009e8)
- Follow-ups outside this folder: none

## v1.4 — 2026-11-02

Kết quả lần chạy spike thứ 3: giới hạn chờ duyệt đã nâng được; SIGINT không hủy được turn mà làm `claude` thoát.

- requirements.md: R-12 viết lại (hủy = SIGTERM/SIGKILL process group, rồi `--resume`); R-13 thêm dọn process con sót; R-16 bỏ marker A-7; bỏ marker và dòng Q-7
- design.md: mục hủy turn dùng process group và Stop; state machine `running --> stopped: Stop`; bảng state đổi Interrupt thành Stop; test plan thêm `TestStopEndsRunningTurn`, `TestStopKillsChildProcesses`, `TestExitReapsLeftoverChildren`
- tasks.md: Status D ghi lần 3; tick R-16; đổi item Interrupt thành Stop; T12 mới chờ chạy lại; bỏ item Q-7
- gherkin/agent-session.feature: scenario R-12 viết lại (dừng, không còn lệnh sót, resume nhớ hội thoại)
- assumptions.md / tech-debt.md: A-7 confirmed; thêm TD-7 (Windows chưa có process group)
- Resolved: Q-7 → không còn áp dụng: hủy giờ kết thúc process nên prompt xếp hàng bị bỏ theo, runner không giữ bộ đếm (Đức chọn "Tắt process rồi resume", 2026-11-02)
- Upstream: claude 2.1.316
- Reason: report lần 3 `docs/spike-report.md` (Đức, 2026-11-02): T7 PASS, T12 FAIL (SIGINT → exit 0, không `result`)
- Follow-ups outside this folder: none

## v1.3 — 2026-11-02

Kết quả lần chạy spike thứ 2 trên máy Đức: inbox không còn bị bypass; phát hiện Claude Code chỉ chờ một lệnh duyệt 60 giây.

- requirements.md: R-10 ghi output deny đã xác nhận; thêm R-16 (`timeout` trong mcp-config), R-17 (approval bị bỏ khi Claude hủy lệnh gọi); SEC-5 bỏ marker A-6; bỏ Q-2, Q-5 khỏi open questions
- design.md: flow duyệt quyền ghi `Abandon` khi lệnh gọi bị hủy và giới hạn chờ của Claude Code; test plan thêm R-16, R-17
- tasks.md: Status D ghi lần 2; tick SEC-5, `approval_prompt`, R-17; thêm item R-16 và T12 chờ chạy lại
- gherkin/agent-session.feature: thêm scenario R-16, R-17
- assumptions.md / tech-debt.md: A-3, A-6 confirmed; thêm A-7; tech-debt no change needed, checked
- Resolved: Q-2 → deny `{behavior, message}` hoạt động, Claude không retry (T6, 2026-11-02); Q-5 → Claude Code mặc định bỏ lệnh gọi sau 60s với "The operation timed out", tool không chạy (T7, 2026-11-02)
- Upstream: claude 2.1.316
- Reason: report lần 2 `docs/spike-report.md` (Đức, 2026-11-02)
- Follow-ups outside this folder: approval-inbox v1.3 (R-19 `Abandon`)

## v1.2 — 2026-11-01

Kết quả lần chạy spike đầu tiên trên máy Đức: trả lời Q-1, Q-3, Q-4, Q-6, phát hiện approval bị bypass bởi settings của owner.

- requirements.md: R-8 viết lại (Claude Code tự xếp hàng, runner đếm `pending`); R-10 ghi input và allow đã xác nhận; R-15 ghi nguồn context và quota; thêm SEC-5 (`permissions.ask`); bỏ marker Q-6; contract surface thêm `tool_result_meta`; thêm Q-7
- design.md: Step 1 bỏ `ErrBusy`; state machine `running` nhận Send, về `idle` khi `pending` = 0; bỏ `ErrBusy` khỏi error mapping; test plan thêm `TestSendWhileRunningQueues`, `TestCheckNotBypassedCatchesUserRule`; stub thêm TD-6
- tasks.md: Status D ghi lần chạy 1; tick gửi khi đang chạy, định danh agent, stub `send_message`, lần chạy 1; thêm item SEC-5 và case Q-7
- gherkin/agent-session.feature: R-8 thêm điều kiện rảnh lại; thêm scenario SEC-5
- assumptions.md / tech-debt.md: A-1, A-4 confirmed; A-3 một phần; thêm A-5, A-6, TD-6; TD-4 cập nhật (bỏ `ForceSend`)
- Resolved: Q-1 → Claude Code xếp hàng, mỗi prompt một `result` (T3, 2026-11-01); Q-3 → context từ `result.modelUsage`, quota chỉ có text `/usage` (T9, T11); Q-4 → path `/mcp/<agent>` + bearer header hoạt động (T15); Q-6 → `system/init` phát ở đầu mỗi turn (T1, T9)
- Upstream: claude 2.1.316
- Reason: report `docs/spike-report.md` do Đức chạy; T5 cho thấy `permission_decision.source = user_permanent`: Bash được rule trong settings của owner duyệt mà không qua `approval_prompt`
- Follow-ups outside this folder: none

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
