# Requirements — storage

Scope: be (orchestrator), Postgres chạy docker compose trên máy owner
Source: ADR-0002; danh sách feature MVP đã duyệt (Đức, 2026-11-05)
Code: chưa có
Mode: Prospective

Orchestrator lưu mọi dữ liệu điều phối (agent, nguồn dữ liệu, project, task, mailbox, approval, event) trong Postgres 17 chạy bằng docker compose trên máy owner. Feature này dựng hạ tầng chung: compose, cấu hình kết nối, migration, transaction, quy ước repository atomic, readiness, và chuyển approval-inbox khỏi in-memory. Bảng của từng feature do feature đó định nghĩa, chạy qua cùng cơ chế migration.

## User stories

- US-007: > As a **user**, I want **tắt bật orchestrator mà không mất agent, task, câu hỏi đang xếp hàng hay approval**, so that **tôi khởi động lại máy mà công việc vẫn còn**.

## Acceptance criteria

### Môi trường local

- R-1: THE SYSTEM SHALL có `compose.yaml` chạy service `postgres` image `postgres:17`, publish port chỉ trên `127.0.0.1:5432`, dữ liệu trong named volume `orch-pgdata`, healthcheck `pg_isready`. (platform)
- R-2: THE SYSTEM SHALL đọc mật khẩu Postgres từ file `.env` (gitignored); repo chỉ commit `.env.example` với giá trị giả. (platform)
- R-3: THE SYSTEM SHALL có target `make db-up`, `make db-down`, `make db-psql`, `make test-integration`. (platform)

### Kết nối và khởi động

- R-4: WHEN orchestrator khởi động, THE SYSTEM SHALL đọc `ORCH_DATABASE_URL` (bắt buộc) và `ORCH_DB_MAX_CONNS` (mặc định 10, 1…50), báo mọi lỗi cấu hình cùng lúc như các biến khác. (platform)
- R-5: WHEN orchestrator khởi động, THE SYSTEM SHALL thử kết nối Postgres trong tối đa `ORCH_DB_CONNECT_TIMEOUT` (mặc định 30s), then:
  - on success: chạy migration (R-6) rồi mới mở HTTP và MCP;
  - on failure (không kết nối được trước hạn): thoát với exit code 1 và log `database unreachable` kèm host và tên DB, không kèm user hay mật khẩu. (platform)
- R-6: WHEN kết nối thành công, THE SYSTEM SHALL áp mọi migration goose còn thiếu trong `migrations/` (embed trong binary), then:
  - on success: log version schema hiện tại;
  - on failure (migration lỗi): rollback migration đó và thoát với exit code 1. (platform)
- R-7: IF schema trong DB mới hơn migration mới nhất trong binary, THEN THE SYSTEM SHALL từ chối khởi động với exit code 1 và log cả hai version. (platform)
- R-8: WHILE orchestrator chạy, THE SYSTEM SHALL trả `GET /readyz` 200 khi ping Postgres thành công trong 2 giây, ngược lại 503 `{"status":"unavailable","dependency":"database"}`. `GET /healthz` không phụ thuộc DB. (platform)

### Repository và transaction

- R-9: THE SYSTEM SHALL cung cấp `TxRunner.WithinTx(ctx, fn)`: mọi repository gọi với ctx đó dùng cùng transaction; `fn` trả lỗi hoặc panic thì rollback, ngược lại commit. (platform)
- R-10: THE SYSTEM SHALL implement `Update(id, mutate)` của mọi repository bằng `SELECT … FOR UPDATE` rồi `UPDATE` trong một transaction, chỉ ghi khi `mutate` trả nil, để hai cập nhật đồng thời trên cùng bản ghi chạy tuần tự (CLAUDE.md rule 3). (platform)
- R-11: IF transaction gặp lỗi serialization hoặc deadlock (SQLSTATE `40001`, `40P01`), THEN THE SYSTEM SHALL chạy lại toàn bộ `fn` tối đa 3 lần với backoff 10/50/250 ms trước khi trả lỗi. (platform)
- R-12: WHEN shutdown, THE SYSTEM SHALL đóng pool sau khi HTTP, MCP và mọi goroutine nền đã dừng. (platform)

### Approval-inbox

- R-13: THE SYSTEM SHALL lưu approval trong bảng `approvals` thay cho repository in-memory, giữ nguyên hành vi approval-inbox R-1…R-11, R-19 (gồm R-7: hai quyết định đồng thời chỉ một thắng). (platform)
- R-14: WHEN orchestrator khởi động lại, THE SYSTEM SHALL chuyển mọi approval còn `pending` sang `expired` với `decided_by = system`, reason "orchestrator khởi động lại", vì không còn agent nào chờ chúng. (platform)

## Security

- SEC-1: THE SYSTEM SHALL không log `ORCH_DATABASE_URL`, user hay mật khẩu; log chỉ có host, port, tên DB.
- SEC-2: THE SYSTEM SHALL chỉ publish port Postgres trên loopback.
- SEC-3: THE SYSTEM SHALL chỉ chạy query có tham số (pgx placeholders); không nối chuỗi giá trị vào SQL.
- SEC-4: Mã hoá at rest dựa vào mã hoá ổ đĩa của máy owner (A-2).

## Contract surface

| Kind | Name | Owner | Notes |
|---|---|---|---|
| File | `compose.yaml`, `.env.example` | this feature | `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` |
| Env var | `ORCH_DATABASE_URL` | this feature | bắt buộc, dạng `postgres://user:pass@127.0.0.1:5432/orch?sslmode=disable` |
| Env var | `ORCH_DB_MAX_CONNS`, `ORCH_DB_CONNECT_TIMEOUT` | this feature | 10; 30s |
| Env var | `ORCH_TEST_DATABASE_URL` | this feature | chỉ cho `make test-integration` |
| Go port | `port.TxRunner` | this feature | |
| HTTP | `GET /readyz` | this feature | |
| Folder | `migrations/` | this feature | goose SQL, mỗi feature thêm file của mình |
| Table | `approvals` | approval-inbox | migration nằm trong feature này |
| Lib | `github.com/jackc/pgx/v5`, `github.com/pressly/goose/v3` | ADR-0002 | |

## Out of scope

- Backup và restore: TD-1.
- Role DB riêng cho ứng dụng thay cho superuser của image: TD-2.
- Deploy lên GCP / Cloud SQL: ngoài GĐ1 (Đức, 2026-11-05: MVP chạy local).
- Retention dữ liệu: roadmap `retention`.

## Open questions

| ID | Câu hỏi | Owner | Liên quan |
|---|---|---|---|
