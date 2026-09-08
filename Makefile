.PHONY: dev down logs build lint test tidy help

# ── Local development ─────────────────────────────────────────────────────────

## dev: Build images and start the full local stack (runs migrations automatically).
dev:
	docker compose up --build

## dev-d: Same as dev but runs in the background (detached).
dev-d:
	docker compose up --build -d

## down: Stop and remove all containers and volumes.
down:
	docker compose down -v

## logs: Tail logs from all services.
logs:
	docker compose logs -f

## logs-proxy: Tail proxy service logs only.
logs-proxy:
	docker compose logs -f proxy

## logs-adminapi: Tail adminapi service logs only.
logs-adminapi:
	docker compose logs -f adminapi

## logs-worker: Tail worker service logs only.
logs-worker:
	docker compose logs -f worker

## ps: Show status of all containers.
ps:
	docker compose ps

# ── Go ────────────────────────────────────────────────────────────────────────

## build: Build all Go binaries locally (not in Docker).
build:
	go build ./cmd/proxy ./cmd/adminapi ./cmd/worker

## tidy: Tidy and verify Go modules.
tidy:
	go mod tidy
	go mod verify

## test: Run all Go unit tests.
test:
	go test -race -count=1 ./...

## lint: Run golangci-lint (requires golangci-lint to be installed).
lint:
	golangci-lint run ./...

## vet: Run go vet.
vet:
	go vet ./...

# ── Utilities ─────────────────────────────────────────────────────────────────

## healthz: Check health of proxy and adminapi.
healthz:
	@echo "→ proxy:   " && curl -sf http://localhost:8080/healthz && echo
	@echo "→ adminapi:" && curl -sf http://localhost:8081/healthz && echo
	@echo "→ dashboard:" && curl -sf http://localhost:3000/healthz && echo

## help: Show this help message.
help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
