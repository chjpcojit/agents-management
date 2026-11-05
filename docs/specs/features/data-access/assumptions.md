# Assumptions — data-access

| ID | Assumption | Basis | Affects | Confirm with | Status |
|---|---|---|---|---|---|
| A-1 | Rule `allow` `WebFetch(domain:x)` thắng `ask` chung của `WebFetch` | Rule cụ thể thường thắng rule chung, chưa có docs khẳng định cho ask/allow | R-8 (`web`) | Q-1 | Open |
| A-2 | Rule đường dẫn tuyệt đối áp được cho thư mục thêm bằng `--add-dir` khi chạy `-p` | Docs settings mô tả `--add-dir` và rule path, chưa thử ở `-p` | R-8 (`dir`) | Q-2 | Open |
| A-3 | `claude` truyền env khai báo trong entry `--mcp-config` (`ORCH_LAUNCH_TOKEN`) cho process MCP stdio | Docs MCP config có field `env` cho server stdio | R-12 | spike T16 chạy kèm | Open |
