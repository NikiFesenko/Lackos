# Changelog

All notable changes to MCP Gate are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

---

## [Phase 1] — 2026-09-08 · Skeleton & Infrastructure

### Added
- **Project structure**: established `cmd/`, `internal/`, `migrations/`, `build/`, `web/` layout.
- **`go.mod`**: module `github.com/mcp-gate/mcp-gate`, Go 1.22.
- **`internal/config`**: shared env-driven config package used by all three binaries; panics clearly on missing required vars (fail-fast).
- **`cmd/proxy`**: skeleton HTTP server on `PROXY_PORT` (default 8080); `GET /healthz` → `{"status":"ok","service":"proxy"}`; graceful SIGTERM shutdown.
- **`cmd/adminapi`**: skeleton HTTP server on `ADMINAPI_PORT` (default 8081); same healthz pattern.
- **`cmd/worker`**: skeleton process that logs readiness and waits for SIGTERM; no consumers yet (wired in Phase 6).
- **`migrations/000001_initial_schema`**: up/down SQL migrations creating `roles`, `users`, `downstream_servers`, `policies`, `audit_events` with all indexes.
- **`docker-compose.yml`**: full local stack — `postgres:16`, `redis:7`, `rabbitmq:3.13-management`, `migrate` (golang-migrate, runs on boot), `proxy`, `adminapi`, `worker`, `dashboard`; all infra services have `healthcheck` blocks; dependent services use `condition: service_healthy` / `service_completed_successfully`.
- **Multi-stage Dockerfiles** for `proxy`, `adminapi`, `worker` (`golang:1.22-alpine` → `distroless/static:nonroot`); `dashboard` uses `node:20-alpine` → `nginx:1.27-alpine`.
- **`web/index.html`**: static placeholder page served by nginx (real React + Vite dashboard in Phase 7).
- **`web/nginx.conf`**: nginx config with SPA routing, `/api/` proxy to adminapi, `/healthz` endpoint.
- **`.env.example`**: all required env vars documented with safe defaults.
- **`Makefile`**: `make dev`, `make down`, `make logs`, `make build`, `make test`, `make healthz`, and more.
- **`README.md`**: project overview and quickstart.

### Design decisions
- **Fail-fast config**: the config package panics rather than silently using zero-values for required vars — a misconfigured container fails immediately and loudly in logs.
- **`migrate` as a service**: migrations run automatically via a dedicated `migrate/migrate` container that exits after completion; application services `depends_on` it completing successfully, so the DB is always in the correct state before any app code runs.
- **`distroless/static:nonroot`** for Go services: no shell, minimal attack surface, runs as non-root by default.
- **`auth_secret_ref` not `auth_secret`**: the schema uses a reference string (e.g. `env:BAMBOOHR_API_KEY`) from day one — the raw secret is never stored in the database.

---

*Next: Phase 2 — Database schema refinement, sqlc query generation, and seed data.*
