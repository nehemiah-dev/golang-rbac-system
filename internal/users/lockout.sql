-- name: IncrementFailedAttempts :one
UPDATE users
SET failed_attempts = failed_attempts + 1
WHERE id = $1
RETURNING failed_attempts;

-- name: LockUser :exec
UPDATE users
SET locked_until = $2
WHERE id = $1;

-- name: GetLockoutStatus :one
SELECT failed_attempts, locked_until
FROM users
WHERE id = $1;

-- name: ResetLockout :exec
UPDATE users
SET failed_attempts = 0, locked_until = NULL
WHERE id = $1;
