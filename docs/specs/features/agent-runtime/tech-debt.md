# Tech debt — agent-runtime

| ID | Debt | Level | Why taken | Impact / risk | Fix | Status |
|---|---|---|---|---|---|---|
| TD-1 | `internal/adapter/runtime/claudecli` chỉ có package doc | High | Skeleton trước, implement trong spike | Chưa chạy được agent | Phase 1–2 | Resolved 2026-11-01 (`feat/agent-runtime-spike`) |
| TD-2 | `internal/adapter/mcpbus` chỉ có package doc | High | Chờ schema permission prompt tool (Q-2) | Chưa duyệt quyền được | Phase 3 | Resolved 2026-11-01 (`feat/agent-runtime-spike`, chuyển sang `internal/delivery/mcpbus`) |
| TD-3 | `send_message` chỉ chuyển cho `MessageSink`, chưa lưu, chưa đánh thức agent nhận | Medium | Office bus là feature sau | Agent tưởng tin đã gửi | Feature office-bus | Open |
| TD-4 | Runner được `cmd/spike` gọi trực tiếp, chưa có port `AgentRuntime` (`ForceSend` đã bỏ ở v1.2) | Medium | Chưa biết hành vi thật để chốt port | Use case chưa dùng được runner | Sau khi chạy lại T5–T7: thêm port | Open |
| TD-5 | `AwaitDecision` poll repository mỗi 250ms | Low | Repository in-memory, chưa có event bus | Trễ tối đa 250ms, tốn CPU nhẹ khi nhiều approval chờ | Notify qua `EventPublisher` của approval-inbox | Open |
| TD-6 | Quota Max lấy bằng parse text của `/usage` | Medium | Không có field có cấu trúc (T11) | Đổi wording ở version `claude` sau là mất số quota, scheduler đoán sai | Parse có test theo version pin; khi không parse được thì hiện nguyên văn và không chặn queue | Open |
| TD-7 | Windows không có process group: Stop chỉ kill `claude`, lệnh con có thể sót | Low | Máy owner là macOS; Windows chưa trong phạm vi | Process mồ côi chạy tiếp trên Windows | Job object (`CreateJobObject` + `KILL_ON_JOB_CLOSE`) | Open |
