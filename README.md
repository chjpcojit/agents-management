# agents-management

Backend orchestrator for managing several Claude Code agents from one place: assign work, stream progress, central approvals, agent-to-agent messaging.

Status: skeleton + agent-runtime spike. See `docs/README.md` for specs and `CLAUDE.md` for architecture rules.

## Run

```
make check
make run
curl localhost:8787/healthz
```

Requires Go 1.25+ and golangci-lint v2.

## Configuration

| Variable | Default | Validation |
|---|---|---|
| `ORCH_HTTP_ADDR` | `127.0.0.1:8787` | `host:port`; must be loopback unless `ORCH_ALLOW_NON_LOOPBACK=true` |
| `ORCH_ALLOW_NON_LOOPBACK` | `false` | boolean; only for use behind Tailscale or similar |
| `ORCH_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `ORCH_APPROVAL_TIMEOUT` | `5m` | Go duration, at least `10s`; unanswered approvals are denied |
| `ORCH_SHUTDOWN_TIMEOUT` | `15s` | Go duration, at least `1s` |

All invalid values are reported together at startup and the process exits with code 1.

## Spike: real `claude -p` on your machine

The spike drives the real `claude` binary through the runner and the MCP approval server. It must run on the owner's machine, logged in with the Max plan (`claude`, then `/login`). The orchestrator never reads or passes Claude credentials; API keys in your shell are stripped from the agent env.

```
make spike-preflight          # claude installed? logged in with Max? API key in env?
make spike-test               # T1–T15 automatically → docs/spike-report.md (~15–25 min)
make spike                    # interactive: type prompts, :allow / :deny approvals
```

| Variable | Default | Meaning |
|---|---|---|
| `SPIKE_CLAUDE` | `claude` | binary name or path |
| `SPIKE_MODEL` | (claude default) | e.g. `sonnet`; a cheaper model saves Max quota |
| `SPIKE_DIR` | OS temp dir | parent folder for throwaway sandboxes |
| `SPIKE_ONLY` | all | e.g. `SPIKE_ONLY=T1,T5` |
| `SPIKE_T7_WAIT` | `2m` | approval deadline in T7 (Q-5: does Claude Code time out first?) |

Raw NDJSON per case goes to `docs/spike-samples/` (gitignored). Review it before sharing.
