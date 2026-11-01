.PHONY: lint test build run check

lint:
	golangci-lint run ./...

test:
	go test -race -count=1 ./...

build:
	go build -o bin/orchestrator ./cmd/orchestrator

run:
	go run ./cmd/orchestrator

check: lint test build

# Spike: drives the real `claude` on the owner's machine (Max login). See docs/specs/features/agent-runtime.
# Spike: chạy `claude` thật trên máy owner (login Max). Xem docs/specs/features/agent-runtime.
SPIKE_CLAUDE ?= claude
SPIKE_MODEL  ?=
SPIKE_DIR    ?=
SPIKE_ONLY   ?=
SPIKE_T7_WAIT ?= 2m

.PHONY: spike-preflight spike spike-test

spike-preflight:
	go run ./cmd/spike preflight -claude=$(SPIKE_CLAUDE)

spike:
	go run ./cmd/spike run -claude=$(SPIKE_CLAUDE) -model=$(SPIKE_MODEL) -dir=$(SPIKE_DIR) -raw=docs/spike-samples/interactive.ndjson

spike-test:
	go run ./cmd/spike test -claude=$(SPIKE_CLAUDE) -model=$(SPIKE_MODEL) -dir=$(SPIKE_DIR) -only=$(SPIKE_ONLY) -t7-wait=$(SPIKE_T7_WAIT)
