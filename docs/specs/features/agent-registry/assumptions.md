# Assumptions — agent-registry

| ID | Assumption | Basis | Affects | Confirm with | Status |
|---|---|---|---|---|---|
| A-1 | Bảng `approvals` chưa có dữ liệu thật khi migration của `agents` chạy, nên đổi `agent_id` sang FK không cần chuyển dữ liệu | storage và agent-registry được làm liền nhau, trước khi chạy agent thật | migration `agents` | thứ tự merge | Open |
