-- +goose Up
CREATE TABLE data.messages (
    id BIGSERIAL PRIMARY KEY,
    uuid TEXT NOT NULL DEFAULT utils.nanoid(8) UNIQUE,
    session_id BIGINT NOT NULL REFERENCES data.sessions(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('system', 'user', 'assistant')),
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_messages_session_id ON data.messages(session_id);
CREATE INDEX idx_messages_uuid ON data.messages(uuid);

-- +goose Down
DROP TABLE IF EXISTS data.messages;
