# Tasks — event-schema

## Status

| Item | Value |
|---|---|
| Overall | Spec only |
| Spec | v1.0 |
| Profile | be |
| Design | design.md |
| A — static | not run |
| B — tests | not run |
| C — driven | not yet |
| D — owner check | not yet |
| E — e2e | not written |
| Branch / commit | — |
| Open | A-1, A-2, TD-1, TD-2 |
| Knowledge | — |

Verification method: A `make lint`, B `make test` (+ `make test-integration` cho Postgres), C `make run` + `curl -N localhost:8787/events`.

## Phase 1 — Contract và domain

- [ ] [be][full] `api/events/v0.schema.json` cho envelope và mọi type trong danh mục — verify: (B) `TestEventsMatchSchema`
- [ ] [be][full] `domain.Event`, `EventType`, `NewEvent` từ chối type ngoài danh mục — verify: (B) `TestNewEvent_Envelope`
- [ ] [be][full] `domain.ClassifyResources` cho tool có cấu trúc và heuristic `Bash` — verify: (B) `TestClassifyResources`

## Phase 2 — Port và adapter

- [ ] [be][full] Port `EventStore`, `EventPublisher` — verify: (A) `make lint`
- [ ] [be][full] Migration bảng `events` + `EventStore` Postgres — verify: (B) `TestDurableEventRollsBackWithState` (integration)
- [ ] [be][full] `eventbus/memory`: ring buffer `ORCH_EVENT_BUFFER`, id theo lô, subscriber 256 — verify: (B) `TestPublisherOrderPerAgent`, `TestLiveEventNotStored`

## Phase 3 — Delivery

- [ ] [be][full] SSE `/events` với filter, `Last-Event-ID`, `stream.gap`, heartbeat, drop subscriber chậm — verify: (B) `TestSSEStreamAndFilter`, `TestSSEReplayAndGap`, `TestSSESlowSubscriberDropped`; (C) curl `-N`
- [ ] [be][full] Chuyển event của approval-inbox sang envelope v0 — verify: (B) test SSE của approval-inbox chạy lại xanh

## Deferred

- Phát `tool.*`, `turn.*`, `stream.unknown` từ runtime: [agent-supervisor](../agent-supervisor/tasks.md).
- Dọn bảng `events`: roadmap `retention`.
