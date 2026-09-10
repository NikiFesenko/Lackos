-- Migration: 000003_create_downstream_servers
-- A DownstreamServer is any MCP-compliant server that MCP Gate proxies to.
-- Credentials are NEVER stored in plaintext. auth_secret_ref is a reference
-- string resolved at runtime (e.g. "env:BAMBOOHR_API_KEY").

CREATE TABLE downstream_servers (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT        UNIQUE NOT NULL,
    base_url        TEXT        NOT NULL,
    auth_type       TEXT        NOT NULL CHECK (auth_type IN ('api_key', 'oauth2')),
    -- Reference into a secrets store — NEVER the raw secret value.
    -- Format examples: "env:BAMBOOHR_API_KEY", "vault:secret/bamboohr#api_key"
    auth_secret_ref TEXT        NOT NULL,
    -- Cached MCP tool manifest: array of {name, description, inputSchema}.
    -- Populated automatically when the server is registered.
    tool_manifest   JSONB       NOT NULL DEFAULT '[]',
    is_active       BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
