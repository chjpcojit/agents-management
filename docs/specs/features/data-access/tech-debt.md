# Tech debt — data-access

| ID | Debt | Level | Why taken | Impact / risk | Fix | Status |
|---|---|---|---|---|---|---|
| TD-1 | [quick] Không cách ly OS: `Bash` của agent đọc được mọi file và env process mà user OS đọc được | High | Sandbox / container cần spike riêng; MVP dựa vào duyệt mọi `Bash` | Owner duyệt nhầm một lệnh là lộ dữ liệu ngoài grant, kể cả secret trong env của MCP server. accepted by Đức, 2026-11-05 (MVP local, mọi Bash qua inbox) | Spike sandbox của Claude Code hoặc container cho mỗi agent; owner Đức; due 2027-01-31 | Open |
| TD-2 | [wait] Không hỗ trợ MCP server HTTP cần header secret | Medium | Header đi qua `--mcp-config` của `claude`, trái SEC-1; cần proxy trong orchestrator | Nguồn chỉ có MCP HTTP (vd SaaS) chưa cấp được | Proxy MCP `/mcp-proxy/<agent>/<source>` gắn header phía orchestrator; owner Đức; due 2027-03-31 | Open |
