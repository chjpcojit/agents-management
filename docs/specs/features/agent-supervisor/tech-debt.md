# Tech debt — agent-supervisor

| ID | Debt | Level | Why taken | Impact / risk | Fix | Status |
|---|---|---|---|---|---|---|
| TD-1 | [wait] Chặn số turn bằng đếm `num_turns` thay vì `--max-turns` | Low | Chưa rõ ngữ nghĩa flag với stdin giữ mở (Q-1) | Agent có thể vượt giới hạn trong turn cuối trước khi bị dừng | Item "Dùng `--max-turns`" trong tasks.md; owner Đức; due 2026-12-31 | Open |
