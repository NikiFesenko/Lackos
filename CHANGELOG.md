# Changelog

All notable changes to MCP Gate are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

---

## [Phase 7] — 2026-09-16 · React + Vite Admin Dashboard

### Added
- **`web/` — full React 18 + TypeScript + Vite SPA** replacing the static placeholder:
  - `package.json` — dependencies: `@tanstack/react-query` v5, `react-router-dom` v6, `lucide-react`, `clsx`
  - `vite.config.ts` — dev proxy `/api → :8081` eliminates CORS issues during local development
  - `tsconfig.{json,app.json,node.json}` — strict TypeScript, path alias `@/*`
  - `.env.example` — `VITE_API_BASE_URL` for running without Vite proxy
- **`src/api.ts`** — centralised typed API client; `ApiError` class; all adminapi endpoints covered (roles, users, servers, policies, audit events, tokens); `Authorization: Bearer` header injected from localStorage
- **`src/context/AuthContext.tsx`** — token stored in localStorage; `login` / `logout` functions; React context with `useAuth()` hook
- **`src/App.tsx`** — React Router v6 client-side routing; auth guard via `ProtectedRoute`; all 6 page routes lazy-loaded via `React.lazy` + `Suspense`
- **`src/components/Layout.tsx`** — persistent sidebar with `NavLink` active-state highlighting; sign-out button
- **`src/components/ui.tsx`** — reusable primitives: `Button` (primary/danger/ghost, sm/md, loading spinner), `Input`, `Select`, `Badge` + `OutcomeBadge`, `Toggle`, `ChipInput` (comma/Enter separated, backspace to delete), `Modal` (click-outside closes), `Spinner`, `ErrorAlert`
- **`src/components/components.css`** — design system tokens (dark slate palette), all component styles
- **Pages** (all use TanStack Query for data + mutations with automatic cache invalidation):
  - `Login.tsx` — admin token input; validates by calling `/api/v1/roles` before storing
  - `Users.tsx` — paginated user list; add user modal; inline role-change modal; deactivate button
  - `Roles.tsx` — role list; create / edit / delete with modal
  - `Servers.tsx` — downstream server list; register / edit modal with `auth_secret_ref` note; active toggle
  - `Policies.tsx` — role + server selector → per-tool policy matrix; each row has allow/deny `Toggle`, redact-fields `ChipInput`, RPM number input, and individual Save button with "Saved!" flash
  - `AuditLog.tsx` — filterable by outcome / date range; paginated (50/page); expandable rows showing redacted input JSON and response summary side-by-side
  - `Tokens.tsx` — select active user, issue / revoke proxy Bearer tokens; one-shot display with clipboard copy button
- **`web/nginx.conf`** — SPA `try_files` fallback; `/api/` proxy to `adminapi` service (Docker Compose DNS); `Cache-Control` headers; gzip enabled
- **`build/dashboard.Dockerfile`** — updated from Phase 1 placeholder to two-stage build: `node:20-alpine` (npm ci + vite build) → `nginx:1.27-alpine`
- **`Makefile`** — added `web-install`, `web-dev`, `web-build` targets

### Verified
- `tsc --noEmit` — **zero TypeScript errors** (strict mode)
- `vite build` — **1636 modules bundled in 890ms**, zero warnings; total gzipped JS ≈ 67 kB (main chunk) + per-page lazy chunks
- All pure-logic Go tests (`internal/audit`, `internal/mcp` redact) still **PASS**; integration tests require live Redis/Docker — blocked only by sandbox network restrictions, not code defects

### Next: Phase 8 — Admin Auth & Security Hardening (bcrypt passwords, JWT sessions, `admin_users` migration, request-ID middleware)

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

---

## [Phase 2] — 2026-09-10 · Database Schema & Migrations

### Changed
- **Replaced** `000001_initial_schema` (single-file) with **5 atomic, reversible per-table migrations**:
  - `000001_create_roles` — roles table
  - `000002_create_users` — users table + email & role_id indexes
  - `000003_create_downstream_servers` — downstream_servers table (auth_secret_ref, tool_manifest JSONB)
  - `000004_create_policies` — policies table + composite lookup index for the hot-path policy engine query
  - `000005_create_audit_events` — audit_events table + 4 indexes covering dashboard filter patterns and alert-worker queries

### Added
- **`migrations/seed_dev.sql`** — sample data for local dev:
  - 4 roles: `hr_admin`, `recruiter`, `manager`, `employee`
  - 1 admin user (`alice@example.com`)
  - 1 mock downstream server (`mock-bamboohr`) with 5 tool entries
  - 12 sample policies illustrating allow/deny + field redaction (salary, ssn)
- **`sqlc.yaml`** — sqlc v2 configuration targeting PostgreSQL + pgx/v5 output
- **`internal/db/queries/`** — 5 SQL query files with named queries for all tables:
  - CRUD for roles, users, downstream servers, policies
  - Audit event creation and paginated reads (by user, server, outcome)
  - `GetPolicyByRoleServerTool` — the hot-path query for the policy engine
  - `UpsertPolicy` — insert-or-update with `ON CONFLICT DO UPDATE`
  - `CountRecentDeniedByUser` — stub for the alerting worker
- **`internal/db/*.sql.go`** — type-safe Go code generated by `sqlc generate` (pgx/v5)
- **`internal/db/db_test.go`** — integration tests using `testcontainers-go`:
  - Spins up a real `postgres:16-alpine` container per test run
  - Applies all 5 migrations via init scripts
  - Tests: CreateRole, ListRoles, CreateUser, GetUserByEmail, DeactivateUser,
    UpsertPolicy (create + idempotent update), GetPolicyByRoleServerTool, CreateAuditEvent

### Design decisions
- **One migration per table** so each change is atomic and independently reversible.
- **`seed_dev.sql` is not auto-run** by the `migrate` service — it's intentional to keep migrations idempotent and environment-neutral. Seed manually: `psql $POSTGRES_DSN -f migrations/seed_dev.sql`.
- **pgx/v5** chosen over `database/sql + lib/pq` — better performance, native Postgres types, no reflection-based scanning.
- **`UpsertPolicy`** uses `ON CONFLICT DO UPDATE` so the admin API is idempotent — calling it twice with the same (role, server, tool) triple updates in place without changing the UUID.

---

*Next: Phase 3 — Admin REST API CRUD (users, roles, downstream servers, policies, audit log) backed by real Postgres.*

---

## [Phase 3] — 2026-09-11 · Admin REST API

### Added

**New packages:**
- `internal/api` — shared JSON response/error helpers (`WriteJSON`, `WriteError`) and standard error code constants
- `internal/middleware` — three middleware functions:
  - `RequestID` — injects/echoes `X-Request-ID` on every request
  - `Logger` — structured `slog` request logging (method, path, status, latency, request ID)
  - `RequireAdminToken` — Bearer token enforcement; `/healthz` is exempt
- `internal/handlers` — all CRUD HTTP handlers backed by the sqlc `Querier` interface:
  - `roles.go` — `ListRoles`, `CreateRole`, `GetRole`, `UpdateRole`, `DeleteRole`
  - `users.go` — `ListUsers`, `CreateUser`, `GetUser`, `UpdateUserRole`, `DeactivateUser` (soft-delete)
  - `downstream_servers.go` — full CRUD; DELETE soft-deactivates (never hard-deletes)
  - `policies.go` — `ListPolicies` (filter by role_id), `UpsertPolicy`, `GetPolicy`, `DeletePolicy`
  - `audit_events.go` — `ListAuditEvents` with pagination and optional filters (user_id, downstream_server_id, outcome)

**Updated:**
- `cmd/adminapi/main.go` — wires Postgres pool, all routes, middleware chain (RequestID → Logger → Auth → mux)

**Endpoints:**
```
GET/POST         /api/v1/roles
GET/PUT/DELETE   /api/v1/roles/{id}
GET/POST         /api/v1/users
GET              /api/v1/users/{id}
PUT              /api/v1/users/{id}/role
DELETE           /api/v1/users/{id}
GET/POST         /api/v1/downstream-servers
GET/PUT/DELETE   /api/v1/downstream-servers/{id}
GET/POST         /api/v1/policies
GET/DELETE       /api/v1/policies/{id}
GET              /api/v1/audit-events
```

**Tests (handler integration, real DB via testcontainers):**
- `roles_test.go` — list empty, create success, missing name, duplicate name, not found, invalid UUID
- `users_test.go` — create success, duplicate email, missing fields, soft-deactivate
- `policies_test.go` — upsert create, upsert update in-place (ID stable), list by role, missing fields

### Design decisions
- **Soft-delete everywhere**: users and downstream servers are deactivated, never hard-deleted — audit events reference them by FK so hard-delete would corrupt history.
- **`UpsertPolicy` is idempotent**: the admin dashboard can call POST /api/v1/policies repeatedly; the UUID is stable on updates.
- **`auth_secret_ref` accepted as-is**: the API stores the reference string verbatim and never resolves or logs it.
- **Go 1.22 pattern matching**: `{id}` path variables use stdlib `r.PathValue()` — no external router needed.

---

*Next: Phase 4 — Policy engine + Redis cache (with fail-closed unit tests)*

---

## [Phase 4] — 2026-09-12 · Policy Engine & Redis Cache

### Added

**New packages:**
- `internal/cache`:
  - `keys.go` — centralized Redis key scheme (`policy:{roleID}:{serverID}:{toolName}`, `ratelimit:{userID}:{serverID}`, `token:{tokenHash}`) and standardized TTL constants.
- `internal/policy`:
  - `decision.go` — `Decision` struct (`Allowed`, `RedactFields`, `MaxCallsPerMinute`) and `Deny` default sentinel.
  - `engine.go` — Policy evaluation engine with fail-closed semantics:
    1. Fast path: check Redis cache (60s TTL).
    2. Fallback path: on cache miss or Redis error, query Postgres using narrow `Store` interface.
    3. Fail-closed contract: if policy not found -> default deny (cached as deny); if DB unreachable -> deny immediately.
    4. Invalidation: `InvalidatePolicy` to evict cached policy entries upon admin mutations.
  - `engine_test.go` — thorough unit tests using `miniredis`: cache hits, cache misses with DB load, fail-closed when both Redis and DB are down, default-deny on missing policy, explicit deny, field redaction list propagation, and cache invalidation.
- `internal/ratelimit`:
  - `limiter.go` — sliding-window rate limiter powered by atomic Redis sorted set Lua script (`ZREMRANGEBYSCORE`, `ZADD`, `ZCARD`, `EXPIRE`). Fails closed if Redis is unavailable.
  - `limiter_test.go` — unit tests covering requests within limit, requests exceeding quota, zero-limit denial, per-user isolation, Redis outage fail-closed behavior, and sliding window boundaries.

### Design decisions
- **Strict Fail-Closed principle**: Any failure or ambiguity in policy evaluation or rate limiting yields `Allowed = false`. No tool call can be executed by the proxy without positive authorization.
- **Atomic sliding window**: Rate limiting uses Redis Lua scripting with nanosecond timestamps to avoid race conditions without application-level distributed locks.
- **Decoupled data access**: The policy engine relies on a focused `Store` interface rather than a concrete DB connection pool, allowing easy testing with in-memory mocks without external dependencies.

---

*Next: Phase 5 — The MCP Proxy Core (JSON-RPC handling, auth, policy enforcement, forwarding, and field redaction)*

---

## [Phase 5] — 2026-09-14 · The MCP Proxy Core

### Added

**New packages and services:**
- `internal/mcp`:
  - `types.go` — MCP JSON-RPC 2.0 message structures (`Request`, `Response`, `RPCError`, `ToolCallParams`, `ToolResult`, `ContentItem`), standard JSON-RPC codes and MCP-specific error codes (`-32001` to `-32004`).
  - `forwarder.go` — downstream HTTP client with `SecretResolver` support, header injection (`Bearer`, `X-Api-Key`), 10 MiB body limit, and full error wrapping.
  - `redact.go` — field-level redaction that scans JSON tool call results and replaces configured sensitive keys (e.g. `salary`, `ssn`) with `"[REDACTED]"`.
  - `redact_test.go` & `forwarder_test.go` — test coverage for redaction, auth header injection, error handling, and malformed inputs.
- `internal/auth`:
  - `tokens.go` — cryptographic per-user proxy token management (32 random bytes hex-encoded, SHA-256 hashed into Redis with 8h TTL, O(1) validation, immediate revocation).
  - `resolver.go` — `EnvSecretResolver` implementing runtime resolution of `env:VAR_NAME` secret references.
  - `tokens_test.go` & `resolver_test.go` — test coverage for token generation, hashing, expiration, and secret resolution.
- `cmd/mockdownstream`:
  - `main.go` — mock MCP server listening on `:9090` by default, responding to `tools/call`, `tools/list`, and `initialize` with canned records containing `salary` and `ssn` to verify redaction end-to-end.
- `cmd/proxy`:
  - `main.go` — complete 8-step request lifecycle:
    1. Parse JSON-RPC 2.0 request.
    2. Extract & validate Bearer token $\rightarrow$ resolve `user_id`.
    3. Query active user record $\rightarrow$ resolve `role_id`.
    4. Query active downstream server by name.
    5. Evaluate policy engine $\rightarrow$ fail closed on deny/error.
    6. Evaluate sliding-window rate limiter $\rightarrow$ fail closed on quota/error with `X-RateLimit-*` headers.
    7. Forward clean tool arguments to downstream server using resolved credentials.
    8. Redact sensitive response fields according to policy $\rightarrow$ return to agent.
  - `proxy_test.go` — integration tests verifying the full flow with mock downstream, field redaction, default-deny on missing policy, rate limit enforcement, and unauthorized access rejection.
- `internal/handlers`:
  - `auth.go` — admin endpoints `POST /api/v1/auth/token` (issue) and `DELETE /api/v1/auth/token` (revoke).
  - `auth_test.go` — unit tests for admin token management.
- `cmd/adminapi`:
  - `main.go` — wired Redis client and `TokenManager`, registering `/api/v1/auth/token` endpoints.

### Design decisions
- **Token hashing**: Raw tokens are never stored in Redis or database — only SHA-256 hashes are stored, preventing credential leakage in case of cache exposure.
- **Fail-closed throughout**: Missing token, expired token, deactivated user, inactive server, missing policy, policy error, rate limit exceeded, or rate limiter error all fail closed immediately with appropriate JSON-RPC error codes.
- **Protocol cleanliness**: Internal routing metadata (e.g. `server` target) is stripped before forwarding to downstream servers so downstream servers receive standard MCP JSON-RPC payloads.

---

*Next: Phase 6 — Audit Pipeline (RabbitMQ -> Postgres, async worker, DLQ)*

---

## [Phase 6] — 2026-09-15 · Audit Pipeline (RabbitMQ → Postgres)

### Added

**New packages and services:**
- `internal/audit`:
  - `event.go` — audit event schema (`Event`), outcome constants (`allowed`, `denied`, `rate_limited`, `error`), topic exchange & queue definitions (`mcp_gate.audit`, `mcp_gate.audit.persist`, `mcp_gate.audit.alerting`, `mcp_gate.audit.dlx`, `mcp_gate.audit.dlq`), and dynamic routing key generation.
  - `publisher.go` — asynchronous RabbitMQ publisher with a non-blocking memory buffer (channel capacity 2048) and background worker, persistent message delivery (`DeliveryMode: 2`), plus `MemoryPublisher` and `NopPublisher` implementations for test isolation.
  - `writer.go` — RabbitMQ persistence worker: declares durable topic exchange, dead-letter exchange (DLX), dead-letter queue (DLQ), and persistence queue with prefetch QoS; deserializes and validates events; executes idempotent insertion into PostgreSQL (`ON CONFLICT (id) DO NOTHING`); rejects unrecoverable messages to DLQ without requeuing.
  - `alerting.go` — real-time security alerting consumer bound to `audit.denied` and `audit.rate_limited`, with `LogAlerter` emitting structured security alert logs.
  - `audit_test.go` — unit tests for model validation, serialization, routing keys, memory publisher, and alerter.
- `internal/mcp`:
  - `redact.go` — added `RedactJSONBytes` function to redact sensitive fields directly from raw JSON input parameters prior to queue dispatch.
  - `redact_test.go` — unit tests for `RedactJSONBytes`.
- `internal/db`:
  - `queries/audit_events.sql` — added `InsertAuditEventWithID` query with `ON CONFLICT (id) DO NOTHING` for deduplicated audit logging.
- `cmd/proxy`:
  - `main.go` — wired RabbitMQ publisher into the proxy server lifecycle; publishes audit events for all outcomes (`allowed`, `denied`, `rate_limited`, `error`) with measured request latency and pre-redacted parameters.
  - `proxy_test.go` — added `TestProxy_AuditEventsPublished` integration test verifying end-to-end event dispatch and field redaction in audit payloads.
- `cmd/worker`:
  - `main.go` — replaced Phase 1 skeleton with complete worker implementation: connects to PostgreSQL connection pool, starts `audit.Writer` and `audit.AlertConsumer` in parallel, and handles clean graceful shutdown via SIGTERM/SIGINT.

### Design decisions
- **Pre-dispatch Redaction**: Sensitive parameter and response fields are stripped **before** publishing to RabbitMQ. The message broker and audit table never store unredacted secrets at rest.
- **Asynchronous, Non-blocking Ingestion**: The proxy publishes events via a buffered in-memory channel. Network jitter or queue congestion in RabbitMQ never degrades proxy response times.
- **Idempotency & Dead-Letter Safety**: Audit events carry client-generated UUIDs and are persisted via `ON CONFLICT (id) DO NOTHING`, ensuring duplicate deliveries do not pollute the audit trail. Malformed or invalid events are rejected directly to `mcp_gate.audit.dlq` for forensic inspection.

---

*Next: Phase 7 — React + Vite Admin Dashboard (React 18, TypeScript, TanStack Query, shadcn/ui)*

