# MCP Gate

> A self-hostable governance proxy that sits between AI agents and MCP servers for HR/people-ops tools.

Many community-built MCP servers authenticate with a single shared API key — no per-user identity, no permission scoping, no audit trail. **MCP Gate** fixes this by requiring every request to authenticate as a specific individual employee, enforcing role-based policies (including field-level response redaction), and logging every tool call asynchronously.

```
AI Agent ──▶ MCP Gate Proxy ──▶ Downstream MCP Server (BambooHR, etc.)
                 │
                 ├── Authenticates per-user
                 ├── Enforces policy (Redis-cached, fail-closed)
                 ├── Applies response redaction
                 └── Publishes audit event ──▶ RabbitMQ ──▶ Postgres
```

## Prerequisites

- [Docker](https://www.docker.com/) 24+
- [Docker Compose](https://docs.docker.com/compose/) v2 (bundled with Docker Desktop)
- `make` (pre-installed on macOS/Linux)

## Quickstart

```bash
# 1. Clone the repo
git clone https://github.com/mcp-gate/mcp-gate.git
cd mcp-gate

# 2. Create your local env file
cp .env.example .env
# Edit .env and set strong passwords before deploying anywhere non-local.

# 3. Start everything (builds images + runs migrations automatically)
make dev
```

That's it. The stack is up when you see all services healthy.

## Service Map

| Service        | Port  | Description                              |
|----------------|-------|------------------------------------------|
| `proxy`        | 8080  | MCP-facing gateway (agent → proxy)       |
| `adminapi`     | 8081  | REST API for the admin dashboard         |
| `dashboard`    | 3000  | React + Vite admin UI                    |
| `postgres`     | 5432  | Primary database                         |
| `redis`        | 6379  | Policy cache & rate limiting             |
| `rabbitmq`     | 5672  | Audit event message queue                |
| RabbitMQ UI    | 15672 | RabbitMQ management console (dev only)   |

## Verify the stack

```bash
make healthz
# → proxy:    {"status":"ok","service":"proxy"}
# → adminapi: {"status":"ok","service":"adminapi"}
# → dashboard:{"status":"ok","service":"dashboard"}
```

## Useful commands

```bash
make dev        # Build + start all services (foreground)
make dev-d      # Same but detached (background)
make down       # Stop + remove containers and volumes
make logs       # Tail all logs
make healthz    # Quick health check of all three app services
make test       # Run Go unit tests
make build      # Build Go binaries locally (not in Docker)
```

## Build phases

| Phase | Status | Description |
|-------|--------|-------------|
| 1 | ✅ Done | Skeleton & infrastructure |
| 2 | ⏳ Next | Database schema, sqlc, seed data |
| 3 | — | Admin REST API CRUD |
| 4 | — | Policy engine + Redis cache |
| 5 | — | MCP proxy core |
| 6 | — | Audit pipeline (RabbitMQ → Postgres) |
| 7 | — | React + Vite admin dashboard |
| 8 | — | Admin auth + security hardening |
| 9 | — | Real downstream integration |
| 10 | — | Production readiness |

## Security notes

- **Fails closed**: if policy cannot be confirmed, requests are denied — never allowed by default.
- **No raw secrets**: downstream credentials are stored as references (e.g. `env:BAMBOOHR_API_KEY`), never as plaintext in the database.
- **Redact before persist**: sensitive fields are stripped before audit events are published to RabbitMQ — they never touch Postgres.
- **Per-user tokens**: agents authenticate with individual tokens, not the shared downstream credential.

## License

MIT
