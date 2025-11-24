-- +migrate Up

CREATE TABLE IF NOT EXISTS ai_jobs (
    id VARCHAR PRIMARY KEY,
    recording_id VARCHAR,
    room_id VARCHAR,
    status VARCHAR NOT NULL,
    attempts INTEGER DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ai_jobs_recording_id ON ai_jobs(recording_id);
CREATE INDEX IF NOT EXISTS idx_ai_jobs_room_id ON ai_jobs(room_id);

-- +migrate Down
-- Rollback: drop ai_jobs table

DROP TABLE IF EXISTS ai_jobs;
