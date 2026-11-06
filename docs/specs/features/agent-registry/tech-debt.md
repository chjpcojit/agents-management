# Tech debt — agent-registry

| ID | Debt | Level | Why taken | Impact / risk | Fix | Status |
|---|---|---|---|---|---|---|
| TD-1 | [quick] Không lưu lịch sử nội dung cấu hình, event chỉ có tên field đổi | Low | Không đưa prompt vào event (SEC-1) | Không xem lại được prompt cũ khi agent đổi hành vi | Bảng `agent_config_versions` lưu snapshot; owner Đức; due 2027-03-31 | Open |
