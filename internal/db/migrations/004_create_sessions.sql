-- +goose Up
CREATE TABLE data.sessions (
    id BIGSERIAL PRIMARY KEY,
    uuid TEXT NOT NULL DEFAULT utils.nanoid(8) UNIQUE,
    user_id BIGINT NOT NULL REFERENCES data.users(id) ON DELETE CASCADE,
    system_prompt TEXT,
    ended_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_user_id ON data.sessions(user_id);
CREATE INDEX idx_sessions_uuid ON data.sessions(uuid);
CREATE INDEX idx_sessions_active ON data.sessions(user_id) WHERE ended_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS data.sessions;
