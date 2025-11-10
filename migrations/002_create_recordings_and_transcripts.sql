-- +migrate Up
-- Migration: 002_create_recordings_and_transcripts
-- Created: 2024-11-05
-- Description: Create tables for recordings and transcriptions

-- ============================================================================
-- RECORDINGS TABLE
-- ============================================================================

CREATE TABLE recordings (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- References
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    started_by UUID REFERENCES users(id),
    
    -- LiveKit Integration
    livekit_recording_id VARCHAR(255) UNIQUE,
    livekit_egress_id VARCHAR(255),
    
    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'recording' CHECK (
        status IN ('recording', 'processing', 'completed', 'failed', 'deleted')
    ),
    
    -- File Info
    file_url TEXT,
    file_path TEXT,
    file_size BIGINT,
    file_format VARCHAR(20) DEFAULT 'mp4',
    
    -- Timing
    duration INT,
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP,
    
    -- Processing
    processing_started_at TIMESTAMP,
    processing_completed_at TIMESTAMP,
    processing_error TEXT,
    
    -- Metadata
    video_tracks INT DEFAULT 0,
    audio_tracks INT DEFAULT 0,
    resolution VARCHAR(20),
    bitrate INT,
    metadata JSONB DEFAULT '{}'::jsonb,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_recordings_room ON recordings(room_id);
CREATE INDEX idx_recordings_status ON recordings(status);
CREATE INDEX idx_recordings_livekit ON recordings(livekit_recording_id) WHERE livekit_recording_id IS NOT NULL;
CREATE INDEX idx_recordings_started ON recordings(started_at DESC);
CREATE INDEX idx_recordings_started_by ON recordings(started_by) WHERE started_by IS NOT NULL;

-- Trigger
CREATE TRIGGER update_recordings_updated_at 
    BEFORE UPDATE ON recordings
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- TRANSCRIPTS TABLE
-- ============================================================================

CREATE TABLE transcripts (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- References
    recording_id UUID NOT NULL REFERENCES recordings(id) ON DELETE CASCADE,
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    
    -- Content
    text TEXT NOT NULL,
    language VARCHAR(10) NOT NULL DEFAULT 'en',
    
    -- Detailed Segments
    segments JSONB NOT NULL DEFAULT '[]'::jsonb,
    
    -- Word-level Timestamps
    words JSONB,
    
    -- Quality Metrics
    confidence_score FLOAT,
    has_speakers BOOLEAN DEFAULT false,
    speaker_count INT DEFAULT 0,
    
    -- Processing Info
    processing_time INT,
    model_used VARCHAR(50),
    
    -- Metadata
    metadata JSONB DEFAULT '{}'::jsonb,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT unique_transcript_per_recording UNIQUE (recording_id)
);

-- Indexes
CREATE INDEX idx_transcripts_recording ON transcripts(recording_id);
CREATE INDEX idx_transcripts_room ON transcripts(room_id);
CREATE INDEX idx_transcripts_language ON transcripts(language);
CREATE INDEX idx_transcripts_segments ON transcripts USING GIN (segments);
CREATE INDEX idx_transcripts_text_search ON transcripts USING GIN (to_tsvector('english', text));

-- Trigger
CREATE TRIGGER update_transcripts_updated_at 
    BEFORE UPDATE ON transcripts
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- +migrate Down
-- Rollback recordings and transcripts tables

-- Drop triggers
DROP TRIGGER IF EXISTS update_transcripts_updated_at ON transcripts;
DROP TRIGGER IF EXISTS update_recordings_updated_at ON recordings;

-- Drop tables
DROP TABLE IF EXISTS transcripts;
DROP TABLE IF EXISTS recordings;
