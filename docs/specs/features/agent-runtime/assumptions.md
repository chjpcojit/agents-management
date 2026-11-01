# Assumptions — agent-runtime

| ID | Assumption | Basis | Affects | Confirm with | Status |
|---|---|---|---|---|---|
| A-1 | Pin một version `claude` không bật bare mặc định cho `-p`, nên login Max vẫn dùng được | Docs headless 2026-11-01: bare "will become the default for -p in a future release" | R-1, R-2 | spike T1 | Open |
| A-2 | Dùng `go.uber.org/goleak` trong test để bắt goroutine leak | Thư viện phổ biến, chỉ dùng trong test | Constraint 5 | — | Open (spike dùng `runtime.NumGoroutine` ở T14 thay thế) |
| A-3 | Permission prompt tool nhận `tool_name`, `input` và trả một text content là JSON `{"behavior":"allow","updatedInput":{...}}` hoặc `{"behavior":"deny","message":"..."}` | Ví dụ trong docs SDK và các bản implement cộng đồng; không có schema chính thức | R-10, `internal/delivery/mcpbus` | Q-2, spike T5, T6 | Open |
| A-4 | Header `Authorization` trong `--mcp-config` (type http) được gửi trên mọi request MCP | Docs MCP config có field `headers` | SEC-4 | Q-4, spike T15 | Open |
