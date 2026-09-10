-- name: CreateDownstreamServer :one
INSERT INTO downstream_servers (name, base_url, auth_type, auth_secret_ref, tool_manifest)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetDownstreamServer :one
SELECT * FROM downstream_servers
WHERE id = $1;

-- name: GetDownstreamServerByName :one
SELECT * FROM downstream_servers
WHERE name = $1;

-- name: ListDownstreamServers :many
SELECT * FROM downstream_servers
ORDER BY name ASC;

-- name: ListActiveDownstreamServers :many
SELECT * FROM downstream_servers
WHERE is_active = TRUE
ORDER BY name ASC;

-- name: UpdateDownstreamServer :one
UPDATE downstream_servers
SET name            = $2,
    base_url        = $3,
    auth_type       = $4,
    auth_secret_ref = $5
WHERE id = $1
RETURNING *;

-- name: UpdateToolManifest :one
UPDATE downstream_servers
SET tool_manifest = $2
WHERE id = $1
RETURNING *;

-- name: SetDownstreamServerActive :one
UPDATE downstream_servers
SET is_active = $2
WHERE id = $1
RETURNING *;
