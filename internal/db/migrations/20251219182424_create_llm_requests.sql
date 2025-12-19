-- +goose Up
CREATE TABLE data.llm_requests (
    id BIGSERIAL PRIMARY KEY,
    uuid TEXT NOT NULL DEFAULT utils.nanoid(8) UNIQUE,
    session_id BIGINT NOT NULL REFERENCES data.sessions(id) ON DELETE CASCADE,
    input_tokens INT NOT NULL,
    output_tokens INT NOT NULL,
    total_tokens INT NOT NULL,
    model TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_llm_requests_session_id ON data.llm_requests(session_id);
CREATE INDEX idx_llm_requests_uuid ON data.llm_requests(uuid);
CREATE INDEX idx_llm_requests_created_at ON data.llm_requests(created_at);

-- +goose Down
DROP TABLE IF EXISTS data.llm_requests;
