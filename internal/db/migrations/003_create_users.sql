-- +goose Up
CREATE TABLE data.users (
    id BIGSERIAL PRIMARY KEY,
    uuid TEXT NOT NULL DEFAULT utils.nanoid(8) UNIQUE,
    telegram_id BIGINT NOT NULL UNIQUE,
    username TEXT,
    first_name TEXT,
    last_name TEXT,
    language_code TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_telegram_id ON data.users(telegram_id);
CREATE INDEX idx_users_uuid ON data.users(uuid);

-- +goose Down
DROP TABLE IF EXISTS data.users;
