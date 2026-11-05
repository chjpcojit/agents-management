# Tech debt — storage

| ID | Debt | Level | Why taken | Impact / risk | Fix | Status |
|---|---|---|---|---|---|---|
| TD-1 | [quick] Không backup volume `orch-pgdata` | Medium | MVP local, dữ liệu điều phối tái tạo được một phần (transcript, worktree vẫn còn) | Mất volume là mất lịch sử task, mailbox, event | Script `pg_dump` hằng ngày vào thư mục owner chọn; owner Đức; due 2027-01-31 | Open |
| TD-2 | [quick] Ứng dụng kết nối bằng superuser do image tạo; bảng `events` chưa bị chặn `UPDATE`/`DELETE` ở mức quyền | Medium | Đơn giản hoá compose | Bug hoặc lệnh sai có thể sửa / xoá event audit | Init script tạo role `orch_app` không phải owner, `REVOKE UPDATE, DELETE ON events`; owner Đức; due 2027-01-31 | Open |
