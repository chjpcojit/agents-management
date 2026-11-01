# agents-management (backend)

Orchestrator that runs and supervises several Claude Code agents on one machine for one user.
Specs live in `docs/specs/features/`; read the feature spec before changing its code.

## Architecture rules

Clean architecture, dependencies point inward only. Enforced by `depguard` in `.golangci.yml` (`make lint`).

| Layer | Path | May import | Holds |
|---|---|---|---|
| Domain | `internal/domain` | stdlib only | entities, invariants, state transitions, sentinel errors |
| Use case | `internal/usecase/<name>` | domain, `internal/usecase/port` | application rules, orchestration of ports |
| Ports | `internal/usecase/port` | domain | interfaces use cases need (repositories, clock, runtime, bus) |
| Adapters | `internal/adapter/...` | domain, port, platform | implementations of ports: storage, `claude` process, MCP server, OS |
| Delivery | `internal/delivery/...` | domain, use cases, platform | HTTP/WebSocket handlers, request decoding, error mapping |
| Platform | `internal/platform/...` | stdlib | config, logging helpers |
| Composition root | `cmd/orchestrator` | everything | wiring only, no rules |

1. Business rules live in domain or use cases, never in handlers or adapters.
2. Use cases receive config values (durations, limits) as constructor arguments, not the config package.
3. Repositories expose atomic `Update(id, mutate)`; no read-then-write across two calls.
4. Errors: domain sentinels matched with `errors.Is`; delivery maps them to status codes; unmapped → opaque 500.
5. Never log tool input, prompts, file contents, credentials or tokens. Log ids, kinds, statuses.
6. The orchestrator never reads, stores or forwards Claude credentials; the `claude` binary manages its own login.
7. Default deny: an approval nobody answers expires as deny. A late decision never unblocks an agent.
8. HTTP binds to loopback unless `ORCH_ALLOW_NON_LOOPBACK=true`.
9. Code comments are bilingual: English line, then Vietnamese line.
10. Tests: table-driven, fakes of ports, `-race`.

## Commands

- `make lint`: golangci-lint (includes layer rules)
- `make test`: `go test -race ./...`
- `make build` / `make run`
- `make check`: all of the above
