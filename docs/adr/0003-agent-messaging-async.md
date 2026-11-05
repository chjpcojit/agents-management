# ADR-0003 — Agent hỏi nhau bất đồng bộ, reply lấy từ `result` cuối turn

Status: Accepted · Date: 2026-11-05 · Decider: Đức

## Context

Nhiều agent có thể cùng hỏi một agent. Agent nhận phải trả lời tuần tự, đúng người hỏi. Có hai cách cho agent hỏi: chờ đồng bộ trong tool call (như approval), hoặc gửi rồi kết thúc turn.

## Decision

1. **Async.** Tool `ask_agent` trả `queued` ngay; turn của người hỏi kết thúc bình thường. Reply được đưa vào mailbox của người hỏi như một item mới và đánh thức họ bằng turn mới.
2. **Một item mailbox = một turn.** Orchestrator chỉ giao item tiếp theo khi agent đã trả `result` của item trước; không ghi nhiều prompt lên stdin cùng lúc trên đường mailbox.
3. **Reply tự động.** Text của `result` cuối turn xử lý item `ask` chính là reply, gắn `correlation_id` của câu hỏi. Agent không phải gọi tool `reply`.

## Consequences

- Không deadlock khi hai agent hỏi nhau; không giữ slot concurrency trong lúc chờ; không vướng timeout MCP.
- Agent hỏi phải tự nối ngữ cảnh khi reply tới ở turn sau; header của item reply nhắc câu hỏi gốc.
- Agent trả lời không chọn được "không trả lời"; mọi `result` đều thành reply.
- Spec chi tiết: [agent-mailbox](../specs/features/agent-mailbox/requirements.md).
