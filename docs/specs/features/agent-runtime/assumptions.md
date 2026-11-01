# Assumptions — agent-runtime

| ID | Assumption | Basis | Affects | Confirm with | Status |
|---|---|---|---|---|---|
| A-1 | Pin một version `claude` không bật bare mặc định cho `-p`, nên login Max vẫn dùng được | Docs headless 2026-11-01: bare "will become the default for -p in a future release" | R-1, R-2 | spike T1 | Open |
| A-2 | Dùng `go.uber.org/goleak` trong test để bắt goroutine leak | Thư viện phổ biến, chỉ dùng trong test | Constraint 5 | — | Open |
