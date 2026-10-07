-- name: CreateUser :one
INSERT INTO users (email, name, password_hash)
VALUES ($1, $2, $3)
RETURNING id;

-- name: GetUserByEmail :one
SELECT id, password_hash
FROM users
WHERE email = $1;

-- name: GetUserInfo :one
-- Everything except the primary key.
SELECT email, name, created_at
FROM users
WHERE id = $1;
