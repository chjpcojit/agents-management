# Assumptions — agent-supervisor

| ID | Assumption | Basis | Affects | Confirm with | Status |
|---|---|---|---|---|---|
| A-1 | Đếm `num_turns` trong `result` đủ để chặn agent chạy quá `max_turns_per_task` | `result` có `num_turns` (spike T1) | `max_turns_per_task` | Q-1 | Open |
| A-2 | `claude` có flag nhận system prompt bổ sung từ file (`--append-system-prompt-file`); nếu không, truyền nội dung qua `--settings` file hoặc chấp nhận argv | CLI reference liệt kê biến thể đọc từ file cho system prompt | SEC-3, LaunchSpec | kiểm `claude --help` của version pin | Open |
| A-3 | Câu hỏi từ agent khác nên được trả lời trong worktree của task gần nhất của người nhận | Người nhận cần đọc code nó đang làm để trả lời | Chọn scope | Đức | Open |
