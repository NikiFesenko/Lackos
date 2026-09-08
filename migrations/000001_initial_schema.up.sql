-- Migration: 000001_initial_schema
-- Creates the full MCP Gate domain model:
--   roles, users, downstream_servers, policies, audit_events

CREATE TABLE roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT UNIQUE NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email        TEXT UNIQUE NOT NULL,
    display_name TEXT NOT NULL,
    role_id      UUID NOT NULL REFERENCES roles(id),
    is_active    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE downstream_servers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT UNIQUE NOT NULL,
    base_url        TEXT NOT NULL,
    auth_type       TEXT NOT NULL CHECK (auth_type IN ('api_key', 'oauth2')),
    -- Reference into a secrets store (e.g. "env:BAMBOOHR_API_KEY").
    -- The raw secret is NEVER stored here.
    auth_secret_ref TEXT NOT NULL,
    -- Cached tool manifest fetched from the downstream MCP server.
    -- Array of {name, description, inputSchema} objects.
    tool_manifest   JSONB NOT NULL DEFAULT '[]',
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE policies (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id              UUID NOT NULL REFERENCES roles(id),
    downstream_server_id UUID NOT NULL REFERENCES downstream_servers(id),
    tool_name            TEXT NOT NULL,
    is_allowed           BOOLEAN NOT NULL DEFAULT FALSE,
    -- Fields to strip from the downstream response before returning to the agent.
    -- e.g. ARRAY['salary', 'ssn']
    redact_fields        TEXT[] NOT NULL DEFAULT '{}',
    max_calls_per_minute INT NOT NULL DEFAULT 30,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (role_id, downstream_server_id, tool_name)
);

CREATE TABLE audit_events (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id               UUID NOT NULL REFERENCES users(id),
    downstream_server_id  UUID NOT NULL REFERENCES downstream_servers(id),
    tool_name             TEXT NOT NULL,
    -- Sensitive params are redacted BEFORE this record is written.
    input_params_redacted JSONB,
    outcome               TEXT NOT NULL CHECK (outcome IN ('allowed', 'denied', 'rate_limited', 'error')),
    -- Configurable: may omit full response payload to avoid duplicating sensitive data at rest.
    response_summary      JSONB,
    latency_ms            INT,
    occurred_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes for common audit log query patterns.
CREATE INDEX idx_audit_events_user_time   ON audit_events (user_id, occurred_at DESC);
CREATE INDEX idx_audit_events_server_time ON audit_events (downstream_server_id, occurred_at DESC);
CREATE INDEX idx_audit_events_outcome     ON audit_events (outcome, occurred_at DESC);
