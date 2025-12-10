-- +migrate Up
-- Update transcripts table to match new entity schema
-- Add meeting_id and raw_data columns

-- Add meeting_id column if not exists
ALTER TABLE IF EXISTS transcripts
ADD COLUMN IF NOT EXISTS meeting_id UUID REFERENCES rooms(id) ON DELETE CASCADE;

-- Add raw_data column if not exists
ALTER TABLE IF EXISTS transcripts
ADD COLUMN IF NOT EXISTS raw_data JSONB DEFAULT '{}'::jsonb;

-- Create indexes on new columns
CREATE INDEX IF NOT EXISTS idx_transcripts_meeting_id ON transcripts(meeting_id);

-- +migrate Down

DROP INDEX IF EXISTS idx_transcripts_meeting_id;
ALTER TABLE IF EXISTS transcripts
DROP COLUMN IF EXISTS meeting_id,
DROP COLUMN IF EXISTS raw_data;

