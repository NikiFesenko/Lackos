-- name: UpsertPolicy :one
-- Used by the admin API to create or update a policy in one call.
INSERT INTO policies (role_id, downstream_server_id, tool_name, is_allowed, redact_fields, max_calls_per_minute)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (role_id, downstream_server_id, tool_name) DO UPDATE
    SET is_allowed           = EXCLUDED.is_allowed,
        redact_fields        = EXCLUDED.redact_fields,
        max_calls_per_minute = EXCLUDED.max_calls_per_minute
RETURNING *;

-- name: GetPolicy :one
SELECT * FROM policies
WHERE id = $1;

-- name: GetPolicyByRoleServerTool :one
-- Hot path — called by the policy engine on every proxied request.
SELECT * FROM policies
WHERE role_id              = $1
  AND downstream_server_id = $2
  AND tool_name            = $3;

-- name: ListPoliciesByRole :many
SELECT * FROM policies
WHERE role_id = $1
ORDER BY downstream_server_id, tool_name;

-- name: ListPoliciesByRoleAndServer :many
SELECT * FROM policies
WHERE role_id              = $1
  AND downstream_server_id = $2
ORDER BY tool_name;

-- name: DeletePolicy :exec
DELETE FROM policies
WHERE id = $1;

-- name: DeletePoliciesByRole :exec
DELETE FROM policies
WHERE role_id = $1;
