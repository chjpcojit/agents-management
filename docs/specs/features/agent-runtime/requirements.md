# Requirements — agent-runtime

Scope: be (orchestrator), process `claude` trên máy owner
Source: spike 0 spec (`/mnt/project-files/specs/spike-go-claude-p.md`), báo cáo kiểm chứng 5 điểm mở (2026-11-01)
Code: `internal/adapter/runtime/claudecli/`, `internal/delivery/mcpbus/`, `cmd/spike/` (spike-grade)
Mode: Prospective

Mỗi agent chạy dưới dạng một process `claude -p` giữ stdin mở, nói chuyện với orchestrator bằng stream-json. Orchestrator đọc event, chuyển yêu cầu quyền vào [approval-inbox](../approval-inbox/requirements.md) qua một MCP tool, và quản lý vòng đời process. Đây cũng là phạm vi của spike 0: chứng minh làm được bằng Go chỉ với interface có docs.

## Glossary

| Term | Nghĩa |
|---|---|
| Runner | Thành phần Go quản lý đúng một process `claude` |
| stream-json | NDJSON trên stdin/stdout (`--input-format` / `--output-format stream-json`) |
| Permission prompt tool | MCP tool mà Claude Code gọi khi cần duyệt (`--permission-prompt-tool`) |

## User stories

- US-003: > As a **user**, I want **giao việc cho một agent và xem nó làm realtime**, so that **tôi điều khiển agent từ một chỗ**.
- US-004: > As a **user**, I want **agent dùng gói Max của tôi mà orchestrator không giữ credential**, so that **không lộ phiên đăng nhập**.

## Acceptance criteria

### Khởi động và auth

- R-1: WHEN orchestrator khởi động một agent, THE SYSTEM SHALL spawn `claude -p --input-format stream-json --output-format stream-json --verbose` với `cwd` là thư mục của agent và không bật `--bare`. (platform)
- R-2: THE SYSTEM SHALL truyền cho process con một env theo allowlist (`HOME`, `PATH`, `USER`, `LOGNAME`, `SHELL`, `TERM`, `LANG`, `LC_*`, `TMPDIR`, `TZ`, `XDG_*`, `CLAUDE_CONFIG_DIR`, biến proxy, biến hệ thống Windows), không có bất kỳ `ANTHROPIC_*`, `CLAUDE_CODE_OAUTH_TOKEN` hay biến chọn provider, để `claude` dùng login subscription của owner. (platform)
- R-3: IF không có dòng stdout nào trong 90 giây sau khi gửi prompt, hoặc có `system/api_retry` với `error = authentication_failed`, THEN THE SYSTEM SHALL đánh dấu agent `failed` với lý do và dừng process. (platform)

### Giao việc và stream

- R-4: WHEN user gửi prompt cho agent đang `idle`, THE SYSTEM SHALL ghi một user message lên stdin của process đó và chuyển agent sang `running`. (platform)
- R-5: WHILE agent đang `running`, THE SYSTEM SHALL phát mọi dòng stdout thành event typed (init, assistant text/tool_use, user tool_result, stream_event delta, result) theo đúng thứ tự, kèm `session_id`. (platform)
- R-6: IF một dòng stdout không parse được hoặc có `type` lạ, THEN THE SYSTEM SHALL giữ dòng thô trong event `unknown` và không dừng stream. (platform)
- R-7: WHEN nhận event `result`, THE SYSTEM SHALL chuyển agent về `idle` và lưu `session_id`, cost, usage của turn. (platform)
- R-8: WHEN user gửi prompt khi agent đang `running`, THE SYSTEM SHALL ghi ngay prompt lên stdin (Claude Code tự xếp hàng và trả một `result` riêng cho mỗi prompt), đếm số prompt chưa có `result`, và chỉ chuyển agent về `idle` khi số đó về 0. (platform)
  > ⚠️ Not confirmed: SIGINT khi còn prompt xếp hàng thì Claude Code bỏ hay giữ các prompt đó — quyết định runner có phải reset bộ đếm khi interrupt. (Q-7)

### Duyệt quyền

- R-9: THE SYSTEM SHALL chạy agent với `--permission-prompt-tool` trỏ vào MCP tool `approval_prompt` của orchestrator và `--strict-mcp-config`. (platform)
- R-10: WHEN `approval_prompt` được gọi, THE SYSTEM SHALL mở một approval trong approval-inbox và chỉ trả lời tool khi approval rời `pending`, then:
  - on `allowed`: trả allow cho Claude Code;
  - on `denied`: trả deny kèm reason;
  - on `expired`: trả deny với reason "không có người duyệt trước hạn".
  Input đã xác nhận ở T8: `tool_name`, `input`, `tool_use_id`. Output allow `{"behavior":"allow","updatedInput":{...}}` đã xác nhận ở T8, T15.
  > ⚠️ Not confirmed: output deny `{"behavior":"deny","message":...}` và Claude Code có timeout tool call khi chờ lâu không — T6, T7 chưa chạy được vì Bash bị rule của owner duyệt thẳng (xem SEC-5). (Q-2, Q-5)
- R-11: IF MCP server không sẵn sàng khi agent khởi động, THEN THE SYSTEM SHALL không spawn agent và báo lỗi readiness. (platform)

### Dừng và khôi phục

- R-12: WHEN user hủy turn đang chạy, THE SYSTEM SHALL gửi SIGINT cho process, chờ event `result`, và giữ process để nhận prompt tiếp. (platform)
- R-13: WHEN process thoát ngoài ý muốn, THE SYSTEM SHALL phát event `agent_exited` có exit code, đánh dấu agent `stopped`, và cho phép khởi động lại bằng `--resume <session_id>` đã lưu. (platform)
- R-14: WHEN orchestrator shutdown, THE SYSTEM SHALL đóng stdin của mọi process, chờ tối đa `ORCH_SHUTDOWN_TIMEOUT`, rồi SIGTERM process còn sống. (platform)

### Lệnh

- R-15: WHEN user gửi `/compact` hoặc `/context`, THE SYSTEM SHALL chuyển nguyên văn thành prompt và phát event kết quả (`compact_boundary` / context usage).
  Context: lấy từ `result.usage` và `result.modelUsage.<model>.contextWindow` (có cấu trúc); `/context` chỉ trả markdown ("Tokens: 28.2k / 1m (3%)") nên chỉ dùng để hiển thị nguyên văn. Quota Max: `/usage` chỉ trả text ("Current session: 45% used · resets …", "Current week (all models): 31% used · resets …"), parse best-effort (A-5, TD-6).

## Security

- SEC-1: THE SYSTEM SHALL không đọc, lưu hay chuyển tiếp credential Claude; `claude` tự quản login. (platform)
- SEC-2: THE SYSTEM SHALL chạy agent với settings deny `Edit`/`Write` vào `.claude/**` và `crossSessionInbound: refuse`, vì `-p` chạy hooks và `.mcp.json` của thư mục kể cả khi chưa trust. (platform)
- SEC-3: THE SYSTEM SHALL không log prompt, tool input hay tool output; event lưu DB thuộc feature audit sau.
- SEC-4: THE SYSTEM SHALL bind MCP server vào loopback, cấp mỗi agent một URL `/mcp/<agent_id>` và một bearer token ngẫu nhiên 256 bit, kiểm token trên mọi request, và trả 401 khi token sai, agent lạ hoặc agent đã unregister. (platform)
- SEC-5: THE SYSTEM SHALL đặt các tool built-in làm thay đổi máy, ra mạng, sinh việc hoặc lên lịch (`AskTools`: `Bash`, `Edit`, `Write`, `WebFetch`, `Task`, `Skill`, `CronCreate`, …) vào `permissions.ask` của `--settings`, để rule `allow` trong `~/.claude/settings.json` của owner không duyệt thẳng thay cho inbox (`ask` thắng `allow`). (platform)
  > ⚠️ Not confirmed: rule `ask` từ `--settings` ở `-p` có chuyển sang `--permission-prompt-tool` không — kiểm ở lần chạy lại T5–T7. (A-6)

## Contract surface

| Kind | Name | Owner | Notes |
|---|---|---|---|
| CLI | `claude -p` flags | Anthropic | [CLI reference](https://code.claude.com/docs/en/cli-reference.md), pin version khi spike |
| Stream | stream-json in/out | Anthropic | [Headless](https://code.claude.com/docs/en/headless.md) |
| MCP tool | `approval_prompt` | this feature | input `tool_name`, `input`, `tool_use_id` (T8); output text JSON `{behavior, updatedInput \| message}`, deny ⚠️ Q-2 |
| MCP tool | `send_message` | feature office-bus (sau) | stub (TD-3): `to`, `text` ≤ 4000 ký tự, trả `queued` |
| Make | `spike-preflight`, `spike`, `spike-test` | this feature | biến `SPIKE_CLAUDE`, `SPIKE_MODEL`, `SPIKE_DIR`, `SPIKE_ONLY`, `SPIKE_T7_WAIT` |
| Env var | `ANTHROPIC_API_KEY` | Anthropic | phải bị loại khỏi env con (R-2) |
| Stream field | `tool_result_meta[].permission_decision` | Anthropic | chưa có docs; `source` + `reason_type` cho biết ai duyệt (T5); spike dùng để phát hiện bypass |

## External facts

Đọc từ code.claude.com ngày 2026-11-01, Agent SDK TS v0.3.191 bundle Claude Code v2.1.191:
- `-p` không bare nạp CLAUDE.md, skills, settings, hooks như interactive; `--bare` không đọc OAuth và docs nói sẽ thành mặc định của `-p` (A-1).
- Credential precedence: `ANTHROPIC_API_KEY` thắng subscription login ở `-p`.
- Agent teams không chạy trong `-p` / SDK.
- SIGTERM: exit 143, turn đang chạy bị bỏ dở; SIGINT kết thúc turn.

## Out of scope

- Office bus (agent nhắn nhau): feature sau, spike chỉ có stub tool.
- Worktree và workspace: feature workspace (sau), quyết định đã chốt ở [ADR-0001](../../../adr/0001-workspace-worktree.md).
- Quota scheduler, DB: feature sau.
- Web UI.

## Open questions

| ID | Câu hỏi | Owner | Liên quan |
|---|---|---|---|
| Q-2 | Output deny của permission prompt tool (input và allow đã xác nhận) | chạy lại T6 | R-10, A-3 |
| Q-5 | MCP tool chờ duyệt vài phút có bị Claude Code timeout không; nếu có thì đó là trần của `ORCH_APPROVAL_TIMEOUT` | chạy lại T7 | R-10 |
| Q-7 | SIGINT khi còn prompt xếp hàng thì các prompt đó còn không | spike (case mới) | R-8 |
