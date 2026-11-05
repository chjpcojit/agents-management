# Assumptions — event-schema

| ID | Assumption | Basis | Affects | Confirm with | Status |
|---|---|---|---|---|---|
| A-1 | Cấp id cho live event bằng `nextval` gom lô 100 đủ nhanh cho stream text | Text delta vài chục event/giây mỗi agent, tối đa vài agent | R-1, R-3, `eventbus/memory` | đo ở level C | Open |
| A-2 | Ring buffer 5000 live event đủ cho web mất mạng vài phút | 2 agent chạy, ~20 event/giây | R-13 | đo ở level C | Open |
