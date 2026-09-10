-- Migration: 000002_create_users
-- Users are individual employees who authenticate with MCP Gate.
-- Each user belongs to exactly one role.

CREATE TABLE users (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email        TEXT        UNIQUE NOT NULL,
    display_name TEXT        NOT NULL,
    role_id      UUID        NOT NULL REFERENCES roles(id),
    is_active    BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Common lookup: resolve email → user record on every inbound request.
CREATE INDEX idx_users_email ON users (email);
-- Common lookup: list all users in a role.
CREATE INDEX idx_users_role_id ON users (role_id);
