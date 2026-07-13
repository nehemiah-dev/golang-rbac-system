-- name: CreateUser :one
INSERT INTO users (email, password_hash)
VALUES ($1, $2)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1;

-- name: VerifyUser :exec
UPDATE users
SET is_verified = true
WHERE id = $1;

-- name: UpdateUserRole :exec
UPDATE users
SET role = $2
WHERE id = $1;