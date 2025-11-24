-- +migrate Up

-- ============================================================================
-- MEETING_SUMMARIES TABLE
-- ============================================================================

CREATE TABLE meeting_summaries (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- References
    room_id UUID NOT NULL UNIQUE REFERENCES rooms(id) ON DELETE CASCADE,
    transcript_id UUID REFERENCES transcripts(id),
    
    -- Summary Content
    executive_summary TEXT NOT NULL,
    
    -- Structured Data
    key_points JSONB DEFAULT '[]'::jsonb,
    decisions JSONB DEFAULT '[]'::jsonb,
    topics JSONB DEFAULT '[]'::jsonb,
    open_questions JSONB DEFAULT '[]'::jsonb,
    next_steps JSONB DEFAULT '[]'::jsonb,
    
    -- Sentiment Analysis
    overall_sentiment FLOAT,
    sentiment_breakdown JSONB,
    
    -- Metrics
    total_speaking_time INT,
    participant_balance_score FLOAT,
    engagement_score FLOAT,
    
    -- Model Info
    model_used VARCHAR(50),
    processing_time INT,
    
    -- Metadata
    metadata JSONB DEFAULT '{}'::jsonb,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_summaries_room ON meeting_summaries(room_id);
CREATE INDEX idx_summaries_transcript ON meeting_summaries(transcript_id) WHERE transcript_id IS NOT NULL;
CREATE INDEX idx_summaries_sentiment ON meeting_summaries(overall_sentiment) WHERE overall_sentiment IS NOT NULL;
CREATE INDEX idx_summaries_created ON meeting_summaries(created_at DESC);

-- ============================================================================
-- ACTION_ITEMS TABLE
-- ============================================================================

CREATE TABLE action_items (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- References
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    summary_id UUID REFERENCES meeting_summaries(id) ON DELETE CASCADE,
    assigned_to UUID REFERENCES users(id) ON DELETE SET NULL,
    created_by UUID REFERENCES users(id),
    
    -- Task Details
    title VARCHAR(500) NOT NULL,
    description TEXT,
    
    -- Classification
    type VARCHAR(50) DEFAULT 'action' CHECK (
        type IN ('action', 'decision', 'question', 'follow_up', 'research')
    ),
    
    -- Priority & Status
    priority VARCHAR(20) DEFAULT 'medium' CHECK (
        priority IN ('low', 'medium', 'high', 'urgent')
    ),
    status VARCHAR(20) DEFAULT 'pending' CHECK (
        status IN ('pending', 'in_progress', 'completed', 'cancelled', 'blocked')
    ),
    
    -- Timing
    due_date DATE,
    estimated_hours FLOAT,
    
    -- Context
    transcript_reference TEXT,
    timestamp_in_meeting INT,
    
    -- External Integration
    clickup_task_id VARCHAR(255),
    clickup_url TEXT,
    external_task_url TEXT,
    
    -- Completion
    completed_at TIMESTAMP,
    completed_by UUID REFERENCES users(id),
    completion_notes TEXT,
    
    -- Metadata
    tags TEXT[],
    metadata JSONB DEFAULT '{}'::jsonb,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_action_items_room ON action_items(room_id);
CREATE INDEX idx_action_items_summary ON action_items(summary_id) WHERE summary_id IS NOT NULL;
CREATE INDEX idx_action_items_assigned ON action_items(assigned_to) WHERE assigned_to IS NOT NULL;
CREATE INDEX idx_action_items_created_by ON action_items(created_by) WHERE created_by IS NOT NULL;
CREATE INDEX idx_action_items_status ON action_items(status);
CREATE INDEX idx_action_items_priority ON action_items(priority);
CREATE INDEX idx_action_items_due_date ON action_items(due_date) WHERE status != 'completed';
CREATE INDEX idx_action_items_type ON action_items(type);
CREATE INDEX idx_action_items_tags ON action_items USING GIN (tags);
CREATE INDEX idx_action_items_assigned_pending ON action_items(assigned_to, status) 
    WHERE status IN ('pending', 'in_progress');

-- ============================================================================
-- PARTICIPANT_REPORTS TABLE
-- ============================================================================

CREATE TABLE participant_reports (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- References
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    participant_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    summary_id UUID REFERENCES meeting_summaries(id),
    
    -- Report Content
    report_content TEXT NOT NULL,
    
    -- Participation Metrics
    speaking_time INT,
    speaking_percentage FLOAT,
    contribution_count INT DEFAULT 0,
    questions_asked INT DEFAULT 0,
    interruptions INT DEFAULT 0,
    
    -- Engagement
    engagement_score FLOAT,
    attention_score FLOAT,
    
    -- Contributions
    key_contributions JSONB DEFAULT '[]'::jsonb,
    
    -- Tasks
    tasks_assigned_count INT DEFAULT 0,
    tasks_created_count INT DEFAULT 0,
    
    -- Detailed Metrics
    metrics JSONB DEFAULT '{}'::jsonb,
    
    -- Metadata
    metadata JSONB DEFAULT '{}'::jsonb,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT unique_room_participant_report UNIQUE (room_id, participant_id)
);

-- Indexes
CREATE INDEX idx_reports_room ON participant_reports(room_id);
CREATE INDEX idx_reports_participant ON participant_reports(participant_id);
CREATE INDEX idx_reports_summary ON participant_reports(summary_id) WHERE summary_id IS NOT NULL;
CREATE INDEX idx_reports_engagement ON participant_reports(engagement_score) WHERE engagement_score IS NOT NULL;

-- +migrate Down
-- Rollback AI summaries and action items tables

-- Drop tables
DROP TABLE IF EXISTS participant_reports;
DROP TABLE IF EXISTS action_items;
DROP TABLE IF EXISTS meeting_summaries;
