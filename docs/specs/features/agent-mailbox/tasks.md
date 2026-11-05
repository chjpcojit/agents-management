# Tasks — agent-mailbox

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
| Open | A-1 |
| Knowledge | — |

Verification method: A `make lint`, B `make test` + `make test-integration`, D hai agent thật cùng hỏi agent thứ ba trên máy owner.

## Phase 1 — Domain

- [ ] [be][full] `MailboxItem`, state machine, `FormatPrompt` — verify: (B) `TestFormatPrompt`, test bảng chuyển trạng thái
- [ ] [be][full] `AskPolicy` — verify: (B) `TestAskPolicy`

## Phase 2 — Persistence

- [ ] [be][full] Migration `mailbox_items`, `agents.mailbox_seq`, index — verify: (B) migrate up (integration)
- [ ] [be][full] `MailboxRepository` với `SKIP LOCKED`, partial unique `delivering` — verify: (B) `TestEnqueueAssignsSeqAndEvent`, `TestClaimFIFO`, `TestClaimOnePerAgent`, `TestRecoverOnStart` (integration)

## Phase 3 — Use case

- [ ] [be][full] `Enqueue`, `Ask` (idempotency, capacity, policy, rejected event) — verify: (B) `TestIdempotentEnqueue`, `TestCapacityExceptReply`, `TestRecipientArchivedOrPaused`, `TestRejectedAskNotStored`
- [ ] [be][full] `ClaimNext`, `Complete`, `Interrupt` — verify: (B) `TestTwoAskersGetOwnReplies`, `TestCompleteAskCreatesReplyAtomically`, `TestErrorResultStillReplies`, `TestCompleteTaskItem`, `TestInterruptRequeuesThenFails`, `TestClaimSkipsPaused`
- [ ] [be][full] `MarkInterrupted` — verify: (B) `TestMarkInterruptedNotRequeued`
- [ ] [be][full] `ExpireDue` sweeper, `CancelQueued`, `RecoverOnStart` — verify: (B) `TestExpireDue`, `TestCancelQueuedRepliesAskers`, `TestReplyToArchivedAsker`
- [ ] [be][full] Không body trong event — verify: (B) `TestEventsHaveNoBody`

## Phase 4 — Delivery

- [ ] [be][full] MCP tool `ask_agent` thay `send_message` (resolve agent-runtime#TD-3) — verify: (B) `TestAskReturnsImmediately`, `TestAskSenderFromToken`
- [ ] [be][full] HTTP mailbox, item, conversation + OpenAPI — verify: (B) `TestMailboxEndpoints`, `TestConversationOrder`
- [ ] [be][full] Chạy thật: `frontend` và `mobile` cùng hỏi `backend` — verify: (D) máy owner

## Deferred

- Kênh chung, broadcast: roadmap `office-bus`.
