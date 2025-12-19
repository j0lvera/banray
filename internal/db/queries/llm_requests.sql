-- name: CreateLLMRequest :one
INSERT INTO data.llm_requests (session_id, message_id, input_tokens, output_tokens, total_tokens, model)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetSessionLLMRequests :many
SELECT * FROM data.llm_requests
WHERE session_id = $1
ORDER BY created_at ASC;

-- name: GetSessionTokenUsage :one
SELECT
    COALESCE(SUM(input_tokens), 0)::INT AS total_input_tokens,
    COALESCE(SUM(output_tokens), 0)::INT AS total_output_tokens,
    COALESCE(SUM(total_tokens), 0)::INT AS total_tokens,
    COUNT(*)::INT AS request_count
FROM data.llm_requests
WHERE session_id = $1;

-- name: GetUserTokenUsage :one
SELECT
    COALESCE(SUM(lr.input_tokens), 0)::INT AS total_input_tokens,
    COALESCE(SUM(lr.output_tokens), 0)::INT AS total_output_tokens,
    COALESCE(SUM(lr.total_tokens), 0)::INT AS total_tokens,
    COUNT(*)::INT AS request_count
FROM data.llm_requests lr
JOIN data.sessions s ON lr.session_id = s.id
WHERE s.user_id = $1;
