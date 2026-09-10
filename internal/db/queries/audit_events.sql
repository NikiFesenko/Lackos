-- name: CreateAuditEvent :one
INSERT INTO audit_events (
    user_id,
    downstream_server_id,
    tool_name,
    input_params_redacted,
    outcome,
    response_summary,
    latency_ms
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetAuditEvent :one
SELECT * FROM audit_events
WHERE id = $1;

-- name: ListAuditEvents :many
-- Paginated, most-recent first. Filtering is handled at the application layer.
SELECT * FROM audit_events
ORDER BY occurred_at DESC
LIMIT $1 OFFSET $2;

-- name: ListAuditEventsByUser :many
SELECT * FROM audit_events
WHERE user_id = $1
ORDER BY occurred_at DESC
LIMIT $2 OFFSET $3;

-- name: ListAuditEventsByServer :many
SELECT * FROM audit_events
WHERE downstream_server_id = $1
ORDER BY occurred_at DESC
LIMIT $2 OFFSET $3;

-- name: ListAuditEventsByOutcome :many
SELECT * FROM audit_events
WHERE outcome = $1
ORDER BY occurred_at DESC
LIMIT $2 OFFSET $3;

-- name: CountRecentDeniedByUser :one
-- Used by the alerting worker to detect suspicious activity patterns.
SELECT COUNT(*) FROM audit_events
WHERE user_id     = $1
  AND outcome     IN ('denied', 'rate_limited')
  AND occurred_at >= now() - ($2::text)::interval;
