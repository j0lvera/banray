-- name: AddMessage :one
INSERT INTO data.messages (session_id, role, content)
VALUES ($1, $2, $3)
RETURNING id;

-- name: GetSessionMessages :many
SELECT id, uuid, session_id, role, content, created_at
FROM data.messages
WHERE session_id = $1
ORDER BY created_at ASC;

-- name: CountSessionMessages :one
SELECT COUNT(*)
FROM data.messages
WHERE session_id = $1;
