# Assumptions — approval-inbox

| ID | Assumption | Basis | Affects | Confirm with | Status |
|---|---|---|---|---|---|
| A-1 | Quét hết hạn mỗi 1 giây là đủ chính xác cho deadline tính bằng phút | Timeout tối thiểu 10s, sai số ≤ 1s chấp nhận được | R-10, `cmd/orchestrator/main.go` | — | Open |
| A-2 | Một user duy nhất, nên `decided_by` là chuỗi tự do tới khi có xác thực | Phạm vi GĐ1: 1 máy, 1 user | R-4, SEC-3 | — | Open |
