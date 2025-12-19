-- name: GetActiveSession :one
SELECT * FROM data.sessions
WHERE user_id = $1 AND ended_at IS NULL
ORDER BY created_at DESC
LIMIT 1;

-- name: CreateSession :one
INSERT INTO data.sessions (user_id, system_prompt)
VALUES ($1, $2)
RETURNING *;

-- name: EndSession :exec
UPDATE data.sessions SET ended_at = NOW() WHERE id = $1;

-- name: GetUserSessions :many
SELECT * FROM data.sessions
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2;
