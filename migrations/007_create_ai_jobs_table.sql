-- +migrate Up
-- Create ai_jobs table for tracking AI processing jobs
CREATE TABLE IF NOT EXISTS ai_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meeting_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    job_type VARCHAR(50) NOT NULL CHECK (job_type IN ('transcription', 'analysis', 'report_gen')),
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'submitted', 'processing', 'completed', 'failed', 'retrying', 'cancelled')),
    external_job_id VARCHAR(255) UNIQUE,
    recording_url TEXT NOT NULL,
    transcript_id UUID REFERENCES transcripts(id) ON DELETE SET NULL,
    
    -- Processing details
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    last_error TEXT,
    
    -- Metadata (JSONB for flexibility)
    metadata JSONB DEFAULT '{}',
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for query performance
CREATE INDEX IF NOT EXISTS idx_ai_jobs_meeting_id ON ai_jobs(meeting_id);
CREATE INDEX IF NOT EXISTS idx_ai_jobs_status ON ai_jobs(status);
CREATE INDEX IF NOT EXISTS idx_ai_jobs_external_id ON ai_jobs(external_job_id);
CREATE INDEX IF NOT EXISTS idx_ai_jobs_transcript_id ON ai_jobs(transcript_id);
CREATE INDEX IF NOT EXISTS idx_ai_jobs_job_type_status ON ai_jobs(job_type, status);
CREATE INDEX IF NOT EXISTS idx_ai_jobs_created_at ON ai_jobs(created_at);

-- +migrate Down

DROP TABLE IF EXISTS ai_jobs;
