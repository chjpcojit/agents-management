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
