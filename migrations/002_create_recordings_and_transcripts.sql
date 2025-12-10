-- +migrate Up

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

-- ============================================================================
-- TRANSCRIPTS TABLE
-- ============================================================================

CREATE TABLE transcripts (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- References (meeting_id, not room_id for Phase 1)
    meeting_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    
    -- Legacy references (optional, for backward compatibility)
    recording_id VARCHAR(255),
    room_id VARCHAR(255),
    
    -- Content
    text TEXT,
    language VARCHAR(20),
    
    -- Detailed Segments
    segments JSONB DEFAULT '[]'::jsonb,
    
    -- Word-level Timestamps
    words JSONB,
    
    -- Quality Metrics
    confidence_score FLOAT,
    has_speakers BOOLEAN DEFAULT false,
    speaker_count INT DEFAULT 0,
    
    -- Processing Info
    processing_time INT,
    model_used VARCHAR(100) DEFAULT 'assemblyai',
    
    -- Raw response from AssemblyAI
    raw_data JSONB DEFAULT '{}'::jsonb,
    
    -- Metadata
    metadata JSONB DEFAULT '{}'::jsonb,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_transcripts_meeting ON transcripts(meeting_id);
CREATE INDEX IF NOT EXISTS idx_transcripts_recording ON transcripts(recording_id);
CREATE INDEX IF NOT EXISTS idx_transcripts_room ON transcripts(room_id);
CREATE INDEX IF NOT EXISTS idx_transcripts_language ON transcripts(language);
CREATE INDEX IF NOT EXISTS idx_transcripts_text_search ON transcripts USING GIN (to_tsvector('english', text));

-- ============================================================================
-- +migrate Down
-- Rollback recordings and transcripts tables

-- Drop tables
DROP TABLE IF EXISTS transcripts;
DROP TABLE IF EXISTS recordings;
