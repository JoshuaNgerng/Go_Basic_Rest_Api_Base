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
SELECT email, name, created_at, updated_at
FROM users
WHERE id = $1;

-- name: UpdateUserInfoById :one
-- Update a single user's information.
UPDATE users
SET
    email = COALESCE(sqlc.narg('email'), email),
    name  = COALESCE(sqlc.narg('name'), name)
WHERE id = sqlc.arg('id')
RETURNING id, email, name, created_at, updated_at;

-- name: DeleteUserById :one
DELETE FROM users
WHERE id = $1
RETURNING id;
