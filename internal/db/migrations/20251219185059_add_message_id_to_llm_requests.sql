-- +goose Up
ALTER TABLE data.llm_requests
ADD COLUMN message_id BIGINT REFERENCES data.messages(id) ON DELETE SET NULL;

CREATE INDEX idx_llm_requests_message_id ON data.llm_requests(message_id);

-- +goose Down
DROP INDEX IF EXISTS data.idx_llm_requests_message_id;
ALTER TABLE data.llm_requests DROP COLUMN message_id;
