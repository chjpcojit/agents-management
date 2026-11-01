# agents-management

Backend orchestrator for managing several Claude Code agents from one place: assign work, stream progress, central approvals, agent-to-agent messaging.

Status: skeleton. See `docs/README.md` for specs and `CLAUDE.md` for architecture rules.

## Run

```
make check
make run
curl localhost:8787/healthz
```

Requires Go 1.24+ and golangci-lint v2.

## Configuration

| Variable | Default | Validation |
|---|---|---|
| `ORCH_HTTP_ADDR` | `127.0.0.1:8787` | `host:port`; must be loopback unless `ORCH_ALLOW_NON_LOOPBACK=true` |
| `ORCH_ALLOW_NON_LOOPBACK` | `false` | boolean; only for use behind Tailscale or similar |
| `ORCH_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `ORCH_APPROVAL_TIMEOUT` | `5m` | Go duration, at least `10s`; unanswered approvals are denied |
| `ORCH_SHUTDOWN_TIMEOUT` | `15s` | Go duration, at least `1s` |

All invalid values are reported together at startup and the process exits with code 1.
