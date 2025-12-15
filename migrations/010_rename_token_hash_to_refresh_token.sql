-- +migrate Up
-- Rename token_hash to refresh_token for clarity
ALTER TABLE sessions RENAME COLUMN token_hash TO refresh_token;

-- Rename the index as well
DROP INDEX idx_sessions_hash;
CREATE INDEX idx_sessions_refresh_token ON sessions(refresh_token);

-- +migrate Down
-- Revert changes
DROP INDEX idx_sessions_refresh_token;
CREATE INDEX idx_sessions_hash ON sessions(token_hash);
ALTER TABLE sessions RENAME COLUMN refresh_token TO token_hash;
