# Requirements — data-access

Scope: be (orchestrator), GCP Secret Manager, process MCP server trên máy owner
Source: yêu cầu của Đức "chúng ta sẽ có các phần dữ liệu ngoài, và cấu hình cho agent access vào" (2026-11-02); ADR-0004
Code: chưa có
Mode: Prospective

Owner đăng ký **nguồn dữ liệu** một lần (thư mục, MCP server, domain web) rồi **cấp quyền** (grant) cho từng agent. Không có grant thì agent không truy cập được. Feature này lưu nguồn và grant, tính ra `AccessPlan` cho mỗi agent và chạy MCP server có secret mà không để secret lọt sang agent. Biến `AccessPlan` thành flags của `claude` là việc của [agent-supervisor](../agent-supervisor/requirements.md).

## Glossary

| Term | Nghĩa |
|---|---|
| Source | Nguồn dữ liệu: `dir` (thư mục), `mcp` (MCP server stdio), `web` (danh sách domain) |
| Grant | Quyền của một agent trên một source: mode `read`/`write`, với `mcp` thêm danh sách tool |
| AccessPlan | Kết quả gộp mọi grant của agent: thư mục thêm, rule allow/ask/deny, entry MCP |
| Launch token | Token dùng một lần để wrapper `mcp-launch` lấy secret từ orchestrator |

## User stories

- US-010: > As a **user**, I want **cho agent này đọc DB nghiệp vụ còn agent kia chỉ đọc thư mục tài liệu**, so that **mỗi agent chỉ thấy dữ liệu nó cần**.
- US-011: > As a **user**, I want **mật khẩu DB và API token không bao giờ lọt vào prompt, log hay tay agent**, so that **lộ phiên agent không lộ hệ thống nghiệp vụ**.

## Acceptance criteria

### Nguồn dữ liệu

- R-1: WHEN client gọi `POST /data-sources` với `name` (`^[a-z][a-z0-9-]{1,31}$`, unique), `kind` và `spec` hợp lệ, THE SYSTEM SHALL lưu source, ghi event `access.source_changed` và trả 201; lỗi validate trả 400 liệt kê mọi field. (platform)
- R-2: WHEN `kind = dir`, THE SYSTEM SHALL yêu cầu `spec.path` là đường dẫn tuyệt đối tồn tại, chuẩn hoá bằng cách giải symlink, và từ chối (400) nếu đường dẫn đã chuẩn hoá là `/`, thư mục home, hoặc nằm trong `~/.ssh`, `~/.gnupg`, `~/.aws`, `~/.config/gcloud`, `~/.claude`, `~/Library/Keychains`, thư mục dữ liệu của orchestrator, hay workspace root. (platform)
- R-3: WHEN `kind = mcp`, THE SYSTEM SHALL yêu cầu `spec.command` (đường dẫn tuyệt đối tới file thực thi), `spec.args`, `spec.tools` (danh sách tool server cung cấp, ≥ 1), `spec.write_tools` (tập con của `tools`) và `spec.secret_env` (map tên biến env → tên tài nguyên Secret Manager `projects/<p>/secrets/<s>/versions/<v>`). (platform)
- R-4: WHEN `kind = web`, THE SYSTEM SHALL yêu cầu `spec.domains` là 1…20 hostname hợp lệ, không có scheme, path hay wildcard ngoài tiền tố `*.`. (platform)
- R-5: WHEN client gọi `PATCH` hoặc `DELETE /data-sources/{id}`, THE SYSTEM SHALL áp thay đổi, ghi event, và với mọi agent có grant trên source đó áp R-11 (thu hồi có hiệu lực ngay). `DELETE` xoá mọi grant của source. (platform)

### Grant

- R-6: WHEN client gọi `PUT /agents/{id}/grants/{source}` với `mode` (`read`/`write`) và, cho `mcp`, `tools` (tập con của `spec.tools`), THE SYSTEM SHALL lưu grant (tạo hoặc thay), ghi event `access.granted`, và áp cho agent ở lần spawn sau; agent đang chạy không được thêm quyền giữa chừng. (platform)
- R-7: WHEN client gọi `GET /agents/{id}/access`, THE SYSTEM SHALL trả 200 danh sách grant và `AccessPlan` hiện tại của agent (không chứa giá trị secret). (platform)
- R-8: THE SYSTEM SHALL tính `AccessPlan` theo bảng ánh xạ ở [design](design.md#ánh-xạ-grant--accessplan); không có grant thì không có thư mục thêm, không có MCP server nguồn, không có domain web được allow (default deny). (platform)
- R-9: THE SYSTEM SHALL xếp tool ghi của nguồn (`Edit`/`Write` trong thư mục `write`, `spec.write_tools` của MCP) vào nhóm ask, để mỗi lần ghi phải qua approval inbox dù grant là `write`. (platform)
- R-10: THE SYSTEM SHALL đặt tool của MCP không có trong grant vào nhóm deny, và không đưa vào cấu hình của agent server MCP của nguồn mà agent không có grant. (platform)

### Thu hồi

- R-11: WHEN client gọi `DELETE /agents/{id}/grants/{source}` (hoặc nguồn bị sửa / xoá theo R-5), THE SYSTEM SHALL xoá grant, ghi event `access.revoked`, và yêu cầu agent-supervisor dừng process của agent trong vòng 5 giây (kể cả khi đang có turn) để lần chạy tiếp theo dùng `AccessPlan` mới. (platform)

### Secret và MCP server

- R-12: WHEN agent-supervisor spawn agent có grant trên nguồn `mcp` có `secret_env`, THE SYSTEM SHALL cấp một launch token (256 bit, dùng một lần, hết hạn sau 60 giây, gắn với agent và source) và đưa vào cấu hình MCP của agent lệnh `orchestrator mcp-launch --source <id>` thay vì lệnh thật. (platform)
- R-13: WHEN `mcp-launch` chạy, THE SYSTEM SHALL đổi launch token lấy giá trị secret qua endpoint loopback, đặt các biến env đó cho process MCP server thật rồi `exec` nó, then:
  - on failure (token sai, đã dùng, hết hạn): `mcp-launch` thoát mã 78 không in secret; agent không có tool của nguồn đó;
  - on failure (Secret Manager không truy cập được, secret không tồn tại hoặc không có quyền): như trên, và orchestrator ghi event `access.source_changed` với `status = secret_unavailable`. (platform)
- R-14: THE SYSTEM SHALL đọc secret từ GCP Secret Manager bằng Application Default Credentials mỗi lần đổi launch token, không cache giá trị trên đĩa và không giữ trong bộ nhớ sau khi trả về. (platform)

### Quan sát

- R-15: WHEN agent gọi tool của một nguồn, THE SYSTEM SHALL phát `tool.requested` với resource `store = source:<name>` (event-schema R-7). (platform)

## Security

- SEC-1: THE SYSTEM SHALL không bao giờ lưu giá trị secret trong Postgres, file, log, event, `--mcp-config`, `--settings`, env hay argv của process `claude` (ADR-0004).
- SEC-2: THE SYSTEM SHALL chỉ cho principal ADC quyền `roles/secretmanager.secretAccessor` trên đúng các secret được tham chiếu (hướng dẫn setup, không tự cấp IAM).
- SEC-3: THE SYSTEM SHALL fail closed: không lấy được secret thì tool của nguồn không có, không chạy server với env thiếu.
- SEC-4: Endpoint đổi launch token chỉ bind loopback, so token bằng so sánh thời gian hằng, xoá token ngay khi dùng.
- SEC-5: `Bash` của agent vẫn đọc được mọi thứ user OS đọc được, kể cả env của process MCP server (qua `ps`); MVP chặn bằng cách mọi lệnh `Bash` phải qua approval inbox (agent-runtime SEC-5). Cách ly cứng là TD-1.

## Contract surface

| Kind | Name | Owner | Notes |
|---|---|---|---|
| HTTP | `POST/GET /data-sources`, `GET/PATCH/DELETE /data-sources/{id}` | this feature | `api/openapi.yaml` |
| HTTP | `PUT/DELETE /agents/{id}/grants/{source}`, `GET /agents/{id}/access` | this feature | |
| HTTP (internal) | `POST /internal/mcp-launch/redeem` | this feature | loopback, chỉ cho `mcp-launch` |
| CLI | `orchestrator mcp-launch --source <id>` | this feature | token qua env `ORCH_LAUNCH_TOKEN`, không qua argv |
| Event | `access.source_changed`, `access.granted`, `access.revoked` | this feature | |
| Go port | `port.SourceRepository`, `port.GrantRepository`, `port.SecretResolver`, `port.LaunchTokens` | this feature | |
| Table | `data_sources`, `agent_grants` | this feature | |
| Lib | `cloud.google.com/go/secretmanager` | ADR-0004 | |
| Claude Code | permission rule `Read(//path/**)`, `WebFetch(domain:…)`, `mcp__<server>__<tool>`, `--add-dir` | Anthropic | [settings](https://code.claude.com/docs/en/settings), cần kiểm Q-1 |

## Out of scope

- MCP server qua HTTP có header secret: TD-2 (MVP chỉ hỗ trợ MCP stdio, hoặc HTTP không secret).
- Cách ly OS (sandbox, container) cho `Bash`: TD-1.
- Tự cấp IAM trên GCP: owner làm tay theo hướng dẫn.
- Deploy trên GCP: ngoài GĐ1.

## Open questions

| ID | Câu hỏi | Owner | Liên quan |
|---|---|---|---|
| Q-1 | Rule `allow` cụ thể `WebFetch(domain:x)` có thắng `ask` chung `WebFetch` của base policy không, hay `ask` luôn thắng? | spike trên máy Đức | R-8, A-1 |
| Q-2 | Đường dẫn trong permission rule có dùng được cho thư mục ngoài `cwd` thêm bằng `--add-dir` ở `-p` không? | spike trên máy Đức | R-8, A-2 |
