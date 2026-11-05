# Tech debt — event-schema

| ID | Debt | Level | Why taken | Impact / risk | Fix | Status |
|---|---|---|---|---|---|---|
| TD-1 | [quick] Bảng `events` không có retention | Medium | MVP chưa dọn dữ liệu (quyết định lưu trữ 2026-11-01) | Bảng lớn dần, replay chậm | Feature `retention`; owner Đức; due 2027-01-31 | Open |
| TD-2 | [quick] Heuristic `Bash` chỉ nhận lệnh phổ biến, không phân tích pipe, subshell, script | Medium | Parse shell đầy đủ tốn công, resource chỉ để hiển thị | Web hiện `unknown` cho nhiều lệnh; không dùng resource để cấp quyền | Mở rộng bảng rule theo log `unknown` thực tế; owner Đức; due 2027-01-31 | Open |
