-- name: CreateUser :one
INSERT INTO users (email, display_name, role_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUser :one
SELECT * FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListActiveUsers :many
SELECT * FROM users
WHERE is_active = TRUE
ORDER BY display_name ASC;

-- name: UpdateUserRole :one
UPDATE users
SET role_id = $2
WHERE id = $1
RETURNING *;

-- name: SetUserActive :one
UPDATE users
SET is_active = $2
WHERE id = $1
RETURNING *;
