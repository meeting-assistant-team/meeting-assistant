-- +migrate Up
-- Create transcript_utterances table for storing speaker segments
-- This allows for detailed speaker diarization and timeline tracking

CREATE TABLE IF NOT EXISTS transcript_utterances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transcript_id UUID NOT NULL REFERENCES transcripts(id) ON DELETE CASCADE,
    speaker VARCHAR(50) NOT NULL,
    text TEXT NOT NULL,
    start_time FLOAT NOT NULL,
    end_time FLOAT NOT NULL,
    confidence FLOAT DEFAULT 0.0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_transcript_utterances_transcript_id ON transcript_utterances(transcript_id);
CREATE INDEX IF NOT EXISTS idx_transcript_utterances_speaker ON transcript_utterances(speaker);
CREATE INDEX IF NOT EXISTS idx_transcript_utterances_start_time ON transcript_utterances(start_time);

-- Add summary and chapters fields to transcripts table
ALTER TABLE transcripts
ADD COLUMN IF NOT EXISTS summary TEXT,
ADD COLUMN IF NOT EXISTS chapters JSONB DEFAULT '[]'::jsonb;

-- +migrate Down

DROP INDEX IF EXISTS idx_transcript_utterances_start_time;
DROP INDEX IF EXISTS idx_transcript_utterances_speaker;
DROP INDEX IF EXISTS idx_transcript_utterances_transcript_id;
DROP TABLE IF EXISTS transcript_utterances;

ALTER TABLE transcripts
DROP COLUMN IF EXISTS summary,
DROP COLUMN IF EXISTS chapters;
