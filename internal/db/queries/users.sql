-- name: GetUserByTelegramID :one
SELECT * FROM data.users WHERE telegram_id = $1;

-- name: CreateUser :one
INSERT INTO data.users (telegram_id, username, first_name, last_name, language_code)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpsertUser :one
INSERT INTO data.users (telegram_id, username, first_name, last_name, language_code)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (telegram_id) DO UPDATE
SET username = EXCLUDED.username,
    first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name,
    language_code = EXCLUDED.language_code,
    updated_at = NOW()
RETURNING *;
