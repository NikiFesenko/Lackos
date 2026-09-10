-- Migration: 000005_create_audit_events
-- Immutable audit log. Written by the worker consumer from RabbitMQ.
-- Sensitive fields are stripped BEFORE the event is published — never stored here.

CREATE TABLE audit_events (
    id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id               UUID        NOT NULL REFERENCES users(id),
    downstream_server_id  UUID        NOT NULL REFERENCES downstream_servers(id),
    tool_name             TEXT        NOT NULL,
    -- Input params with sensitive values already redacted per policy.
    input_params_redacted JSONB,
    outcome               TEXT        NOT NULL
        CHECK (outcome IN ('allowed', 'denied', 'rate_limited', 'error')),
    -- Optional response summary. Full payload is not stored to avoid
    -- duplicating sensitive data at rest (configurable per deployment).
    response_summary      JSONB,
    latency_ms            INT,
    occurred_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Dashboard: filter by user over time.
CREATE INDEX idx_audit_events_user_time
    ON audit_events (user_id, occurred_at DESC);

-- Dashboard: filter by downstream server over time.
CREATE INDEX idx_audit_events_server_time
    ON audit_events (downstream_server_id, occurred_at DESC);

-- Dashboard: filter by outcome (e.g. show only denied/rate_limited).
CREATE INDEX idx_audit_events_outcome
    ON audit_events (outcome, occurred_at DESC);

-- Alert worker: fast lookup of recent denied events per user.
CREATE INDEX idx_audit_events_user_outcome
    ON audit_events (user_id, outcome, occurred_at DESC);
