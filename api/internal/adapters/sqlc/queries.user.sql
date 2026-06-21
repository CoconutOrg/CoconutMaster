-- name: GetUsers :many
SELECT * FROM users;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 LIMIT 1;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1 LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (username, email, password_hash)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateUserById :one
UPDATE users
SET username = $2, email = $3, password_hash = $4
WHERE id = $1
RETURNING *;

-- name: PatchUserRefreshTokenById :one
UPDATE users
SET refresh_token = $2,
    refresh_token_expiration = $3
WHERE id = $1
RETURNING *;

-- name: PatchUserIsVerifiedById :one
UPDATE users
SET is_verified = $2
WHERE id = $1
RETURNING *;

-- name: DeleteUserById :exec
DELETE FROM users WHERE id = $1;