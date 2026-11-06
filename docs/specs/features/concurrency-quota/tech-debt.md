# Tech debt — concurrency-quota

| ID | Debt | Level | Why taken | Impact / risk | Fix | Status |
|---|---|---|---|---|---|---|
| TD-1 | [wait] Chưa phản ứng với rate limit của gói (R-8) | Medium | Chưa biết tín hiệu trong stream (Q-1) | Hết quota giữa chừng: các turn lỗi liên tục, item hết 3 attempt và `failed` | Item "Rate limit backoff" trong tasks.md sau spike T18; owner Đức; due 2026-12-31 | Open |
