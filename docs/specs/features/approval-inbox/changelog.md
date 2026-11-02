# Changelog — approval-inbox

## v1.3 — 2026-11-02

Thêm đường kết thúc approval khi agent bỏ chờ, theo kết quả spike T7 của agent-runtime.

- requirements.md: thêm R-19
- design.md: no change needed, checked (flow nằm ở agent-runtime design, bước Chờ quyết định)
- tasks.md: thêm và tick `Service.Abandon`
- gherkin/approval-decision.feature: thêm scenario R-19
- assumptions.md / tech-debt.md: no change needed, checked
- Upstream: unchanged
- Reason: spike T7 lần 2: Claude Code bỏ lệnh gọi duyệt sau 60s, approval treo `pending`
- Follow-ups outside this folder: none

## v1.2 — 2026-11-01

Hoãn push notification: GĐ1 chỉ duyệt trên web.

- requirements.md: R-13 removed; out of scope thêm push và risk theo `ClassifyRisk`; bỏ Q-2 khỏi open questions
- design.md: `internal/adapter/notify` chuyển sang deferred (TD-4); bỏ dòng test plan của R-13
- tasks.md: bỏ item push, thêm vào Deferred
- gherkin/approval-decision.feature: bỏ scenario R-13
- assumptions.md: no change needed, checked
- tech-debt.md: thêm TD-4
- Resolved: Q-2 → hoãn push, website trước, duyệt qua SSE (Đức, 2026-11-01)
- Upstream: unchanged
- Reason: Đức "tạm thời không cần đến mobile anh sẽ làm website trước"
- Follow-ups outside this folder: none

## v1.1 — 2026-11-01

Chốt API inbox là REST + SSE, thêm criteria cho các endpoint và kênh event.

- requirements.md: R-12 viết lại theo `GET /approvals`; thêm R-14…R-18; contract surface thêm 3 route REST, `/events`, RFC 7807
- design.md: components thêm `api/openapi.yaml`, handlers, `EventPublisher`, `eventbus/memory`; flow Event SSE (steps + diagram); bỏ marker Q-1 ở error mapping; test plan thêm R-12, R-14…R-18
- tasks.md: Phase 4 tách thành 5 item, bỏ blocked by Q-1
- gherkin/approval-decision.feature: thêm scenario R-14…R-18
- assumptions.md / tech-debt.md: no change needed, checked (TD-3 vẫn chặn mở ngoài loopback)
- Resolved: Q-1 → REST + SSE (Đức, 2026-11-01)
- Upstream: unchanged
- Reason: Đức chọn REST + SSE
- Follow-ups outside this folder: tạo `api/openapi.yaml` (nằm trong Phase 4 của feature này)

## v1.0 — 2026-11-01

v1.0 (initial), Mixed: phần domain và use case viết Retroactive từ code đã có, delivery và push viết Prospective.

- requirements.md: R-1…R-13, SEC-1…SEC-4, Q-1, Q-2
- design.md: constraints, components, flow Decide (steps + diagram), state machine, error mapping, test plan
- tasks.md: Phase 1–3 tick theo `make lint` và `make test` ngày 2026-11-01; Phase 4 blocked
- gherkin/approval-decision.feature: scenario cho R-1…R-13, `@wip` cho R-11…R-13
- assumptions.md / tech-debt.md: A-1, A-2, TD-1…TD-3
- Upstream: none (chưa có product spec, nguồn là design doc trong project memory)
- Reason: Đức yêu cầu dựng project clean architecture và tài liệu bằng skill spec
- Follow-ups outside this folder: none
