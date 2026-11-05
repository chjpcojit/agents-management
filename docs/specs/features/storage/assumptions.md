# Assumptions — storage

| ID | Assumption | Basis | Affects | Confirm with | Status |
|---|---|---|---|---|---|
| A-1 | Máy owner có Docker Desktop (hoặc engine tương đương) chạy được `postgres:17` arm64 | Máy owner là macOS darwin/arm64 (spike 2026-11-01) | R-1, R-3 | Đức | Open |
| A-2 | Ổ đĩa máy owner bật FileVault nên volume Postgres được mã hoá at rest | Thực hành chuẩn trên macOS | SEC-4 | Đức | Open |
