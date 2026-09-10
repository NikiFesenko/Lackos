-- Migration: 000004_create_policies
-- A Policy grants or denies a specific role the ability to call a specific
-- tool on a specific downstream server, with optional field-level redaction.

CREATE TABLE policies (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id              UUID        NOT NULL REFERENCES roles(id),
    downstream_server_id UUID        NOT NULL REFERENCES downstream_servers(id),
    tool_name            TEXT        NOT NULL,
    is_allowed           BOOLEAN     NOT NULL DEFAULT FALSE,
    -- Response fields to strip before returning to the agent.
    -- Example: ARRAY['salary', 'ssn', 'bank_account']
    redact_fields        TEXT[]      NOT NULL DEFAULT '{}',
    max_calls_per_minute INT         NOT NULL DEFAULT 30,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Ensures one rule per (role, server, tool) triple.
    UNIQUE (role_id, downstream_server_id, tool_name)
);

-- Hot path: policy engine looks up (role_id, downstream_server_id, tool_name).
CREATE INDEX idx_policies_lookup
    ON policies (role_id, downstream_server_id, tool_name);
