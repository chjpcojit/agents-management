# Tech debt — approval-inbox

| ID | Debt | Level | Why taken | Impact / risk | Fix | Status |
|---|---|---|---|---|---|---|
| TD-1 | Repository in-memory | High | Dựng skeleton trước khi chọn DB | Restart mất mọi approval pending; agent đang chờ sẽ chờ tới timeout phía agent | Repository SQLite với `Update` atomic | Open |
| TD-2 | Chưa có audit log bất biến | High | Chưa có storage | Không truy được ai duyệt gì sau restart | Bảng event append-only (entity Event trong data model) | Open |
| TD-3 | Chưa xác thực client, chỉ dựa vào bind loopback | Critical | Chưa có delivery | Process local bất kỳ trên máy gọi được API khi delivery xuất hiện | Thêm login cho web trước khi mở endpoint decide | Open |
