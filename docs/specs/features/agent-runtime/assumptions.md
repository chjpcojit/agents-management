# Assumptions — agent-runtime

| ID | Assumption | Basis | Affects | Confirm with | Status |
|---|---|---|---|---|---|
| A-1 | Pin một version `claude` không bật bare mặc định cho `-p`, nên login Max vẫn dùng được | Docs headless 2026-11-01: bare "will become the default for -p in a future release" | R-1, R-2 | spike T1 | Confirmed 2026-11-01 (claude 2.1.316, `apiKeySource=none`) |
| A-2 | Dùng `go.uber.org/goleak` trong test để bắt goroutine leak | Thư viện phổ biến, chỉ dùng trong test | Constraint 5 | — | Open (spike dùng `runtime.NumGoroutine` ở T14 thay thế) |
| A-3 | Permission prompt tool nhận `tool_name`, `input` và trả một text content là JSON `{"behavior":"allow","updatedInput":{...}}` hoặc `{"behavior":"deny","message":"..."}` | Ví dụ trong docs SDK và các bản implement cộng đồng; không có schema chính thức | R-10, `internal/delivery/mcpbus` | Q-2, spike T5, T6 | Confirmed 2026-11-02 (allow T8, T15; deny T6 lần 2) |
| A-4 | Header `Authorization` trong `--mcp-config` (type http) được gửi trên mọi request MCP | Docs MCP config có field `headers` | SEC-4 | Q-4, spike T15 | Confirmed 2026-11-01 (T15) |
| A-5 | Text của `/usage` giữ dạng "Current session: N% used · resets …" trong version `claude` đang pin | Output T11, claude 2.1.316 | R-15, TD-6 | test parse theo version | Open |
| A-6 | Rule `ask` trong `--settings` thắng rule `allow` của user settings và ở `-p` được chuyển tới `--permission-prompt-tool` | Docs permissions: deny > ask > allow | SEC-5 | chạy lại T5–T7 | Confirmed 2026-11-02 (T5, T6 lần 2) |
| A-7 | Field `timeout` (ms) của MCP server type http trong `--mcp-config` nâng giới hạn 60s Claude Code chờ một lệnh gọi tool | Schema config trong binary claude 2.1.316 ("Per-server tool-call timeout in milliseconds. Overrides the MCP_TOOL_TIMEOUT environment variable"); chưa có docs công khai | R-16 | chạy lại T7 | Open |
