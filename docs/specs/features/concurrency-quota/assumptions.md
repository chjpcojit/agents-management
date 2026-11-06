# Assumptions — concurrency-quota

| ID | Assumption | Basis | Affects | Confirm with | Status |
|---|---|---|---|---|---|
| A-1 | Item bị trả về vì agent `failed` (không chạy được turn) vẫn tính attempt theo mailbox R-16 | Đơn giản, agent `failed` hiếm và cần owner xử lý | R-4 | Đức | Open |
| A-2 | Hết quota thể hiện qua lỗi trong stream để scheduler phát hiện được | Hành vi `claude` khi bị rate limit có retry và báo lỗi | R-8 | Q-1 | Open |
| A-3 | `claude -p "/usage"` không tốn lượt model, nên không cần slot | Spike T11: lệnh trả text ngay | R-5 | đo ở level D | Open |
