# Database Schema & Design

## Overview

Hệ thống sử dụng PostgreSQL làm database chính với Redis cho caching và session management.

## Entity Relationship Diagram

```
┌─────────────┐         ┌──────────────┐         ┌──────────────┐
│    users    │────┬────│    rooms     │────┬────│  recordings  │
└─────────────┘    │    └──────────────┘     │   └──────────────┘
       │           │           │             │
       │           │           │             │
       ▼           │           ▼             ▼
┌─────────────┐    │    ┌──────────────┐   ┌──────────────┐
│session      │    └───▶│participants  │   │ transcripts  │
│             │         └──────────────┘   └──────────────┘
└─────────────┘                │                   │
                               │                   │
                               ▼                   ▼
                        ┌──────────────┐   ┌──────────────┐
                        │participant_  │   │meeting_      │
                        │reports       │   │summaries     │
                        └──────────────┘   └──────────────┘
                               │                   │
                               ▼                   │
                        ┌──────────────┐           │
                        │action_items  │◄──────────┘
                        └──────────────┘
                               │
                               ▼
                        ┌──────────────┐
                        │notifications │
                        └──────────────┘
```

## Core Tables

### users

**Purpose**: Store user accounts and authentication info

```sql
CREATE TABLE users (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Authentication
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255), -- NULL for OAuth-only users
    oauth_provider VARCHAR(50), -- 'google', 'github', 'microsoft'
    oauth_id VARCHAR(255),
    oauth_refresh_token TEXT ENCRYPTED, -- For refreshing OAuth tokens
    
    -- Profile
    name VARCHAR(255) NOT NULL,
    avatar_url TEXT,
    bio TEXT,
    timezone VARCHAR(50) DEFAULT 'UTC',
    language VARCHAR(10) DEFAULT 'en',
    
    -- Role & Status
    role VARCHAR(20) DEFAULT 'participant' CHECK (role IN ('admin', 'host', 'participant')),
    is_active BOOLEAN DEFAULT true,
    is_email_verified BOOLEAN DEFAULT false,
    
    -- Settings
    notification_preferences JSONB DEFAULT '{"email": true, "push": true, "reports": true}'::jsonb,
    meeting_preferences JSONB DEFAULT '{"auto_join_audio": true, "auto_join_video": false}'::jsonb,
    
    -- Metadata
    last_login_at TIMESTAMP,
    last_active_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexes
CREATE UNIQUE INDEX idx_users_email ON users(LOWER(email));
CREATE INDEX idx_users_oauth ON users(oauth_provider, oauth_id);
CREATE INDEX idx_users_role ON users(role) WHERE is_active = true;
CREATE INDEX idx_users_created ON users(created_at DESC);

-- Trigger for updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

### rooms

**Purpose**: Meeting rooms and their configurations

```sql
CREATE TABLE rooms (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Basic Info
    name VARCHAR(255) NOT NULL,
    description TEXT,
    slug VARCHAR(100) UNIQUE, -- URL-friendly identifier
    
    -- Ownership
    host_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Room Type
    type VARCHAR(20) NOT NULL DEFAULT 'public' CHECK (type IN ('public', 'private', 'scheduled')),
    status VARCHAR(20) NOT NULL DEFAULT 'scheduled' CHECK (status IN ('scheduled', 'active', 'ended', 'cancelled')),
    
    -- LiveKit Integration
    livekit_room_name VARCHAR(255) UNIQUE NOT NULL,
    livekit_room_id VARCHAR(255),
    
    -- Capacity
    max_participants INT DEFAULT 10 CHECK (max_participants BETWEEN 2 AND 100),
    current_participants INT DEFAULT 0,
    
    -- Settings
    settings JSONB DEFAULT '{
        "enable_recording": true,
        "enable_chat": true,
        "enable_screen_share": true,
        "require_approval": false,
        "allow_guests": false,
        "mute_on_join": false,
        "disable_video_on_join": false,
        "enable_waiting_room": false,
        "auto_record": false,
        "enable_transcription": true
    }'::jsonb,
    
    -- Scheduling
    scheduled_start_time TIMESTAMP,
    scheduled_end_time TIMESTAMP,
    
    -- Timing
    started_at TIMESTAMP,
    ended_at TIMESTAMP,
    duration INT, -- seconds, calculated after meeting ends
    
    -- Metadata
    tags TEXT[], -- For categorization
    metadata JSONB DEFAULT '{}'::jsonb,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_rooms_host ON rooms(host_id);
CREATE INDEX idx_rooms_status ON rooms(status);
CREATE INDEX idx_rooms_type ON rooms(type);
CREATE INDEX idx_rooms_scheduled ON rooms(scheduled_start_time) WHERE status = 'scheduled';
CREATE INDEX idx_rooms_slug ON rooms(slug) WHERE slug IS NOT NULL;
CREATE INDEX idx_rooms_livekit ON rooms(livekit_room_name);
CREATE INDEX idx_rooms_tags ON rooms USING GIN (tags);

-- Trigger
CREATE TRIGGER update_rooms_updated_at BEFORE UPDATE ON rooms
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Calculate duration on meeting end
CREATE OR REPLACE FUNCTION calculate_room_duration()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'ended' AND OLD.status = 'active' THEN
        NEW.duration = EXTRACT(EPOCH FROM (NEW.ended_at - NEW.started_at))::INT;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_calculate_room_duration
BEFORE UPDATE ON rooms
FOR EACH ROW EXECUTE FUNCTION calculate_room_duration();
```

### participants

**Purpose**: Track who joined which room and when

```sql
CREATE TABLE participants (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- References
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Role in Room
    role VARCHAR(20) DEFAULT 'participant' CHECK (role IN ('host', 'co_host', 'participant', 'guest')),
    
    -- Participation Status
    status VARCHAR(20) DEFAULT 'invited' CHECK (status IN ('invited', 'joined', 'left', 'removed', 'declined')),
    
    -- Timing
    invited_at TIMESTAMP,
    joined_at TIMESTAMP,
    left_at TIMESTAMP,
    duration INT, -- seconds in meeting
    
    -- Permissions
    can_share_screen BOOLEAN DEFAULT true,
    can_record BOOLEAN DEFAULT false,
    can_mute_others BOOLEAN DEFAULT false,
    is_muted BOOLEAN DEFAULT false,
    is_hand_raised BOOLEAN DEFAULT false,
    
    -- Actions
    is_removed BOOLEAN DEFAULT false,
    removed_by UUID REFERENCES users(id),
    removal_reason TEXT,
    
    -- Metadata
    connection_quality VARCHAR(20), -- 'excellent', 'good', 'poor'
    device_info JSONB, -- Browser, OS, camera/mic info
    metadata JSONB DEFAULT '{}'::jsonb,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT unique_room_user UNIQUE (room_id, user_id)
);

-- Indexes
CREATE INDEX idx_participants_room ON participants(room_id);
CREATE INDEX idx_participants_user ON participants(user_id);
CREATE INDEX idx_participants_status ON participants(status);
CREATE INDEX idx_participants_joined ON participants(joined_at) WHERE joined_at IS NOT NULL;

-- Trigger
CREATE TRIGGER update_participants_updated_at BEFORE UPDATE ON participants
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Calculate participant duration
CREATE OR REPLACE FUNCTION calculate_participant_duration()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.left_at IS NOT NULL AND NEW.joined_at IS NOT NULL THEN
        NEW.duration = EXTRACT(EPOCH FROM (NEW.left_at - NEW.joined_at))::INT;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_calculate_participant_duration
BEFORE UPDATE ON participants
FOR EACH ROW EXECUTE FUNCTION calculate_participant_duration();
```

### recordings

**Purpose**: Store recording metadata and processing status

```sql
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
    file_path TEXT, -- Relative path in MinIO bucket
    file_size BIGINT, -- bytes
    file_format VARCHAR(20) DEFAULT 'mp4',
    
    -- Timing
    duration INT, -- seconds
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP,
    
    -- Processing
    processing_started_at TIMESTAMP,
    processing_completed_at TIMESTAMP,
    processing_error TEXT,
    
    -- Metadata
    video_tracks INT DEFAULT 0,
    audio_tracks INT DEFAULT 0,
    resolution VARCHAR(20), -- e.g., '1920x1080'
    bitrate INT, -- kbps
    metadata JSONB DEFAULT '{}'::jsonb,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_recordings_room ON recordings(room_id);
CREATE INDEX idx_recordings_status ON recordings(status);
CREATE INDEX idx_recordings_livekit ON recordings(livekit_recording_id);
CREATE INDEX idx_recordings_started ON recordings(started_at DESC);

-- Trigger
CREATE TRIGGER update_recordings_updated_at BEFORE UPDATE ON recordings
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

### transcripts

**Purpose**: Store speech-to-text results

```sql
CREATE TABLE transcripts (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- References
    recording_id UUID NOT NULL REFERENCES recordings(id) ON DELETE CASCADE,
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    
    -- Content
    text TEXT NOT NULL,
    language VARCHAR(10) NOT NULL, -- ISO 639-1 code
    
    -- Detailed Segments
    segments JSONB NOT NULL DEFAULT '[]'::jsonb,
    /* Structure:
    [
        {
            "id": 0,
            "start": 0.0,
            "end": 5.5,
            "text": "Hello everyone",
            "speaker": "user_id",
            "speaker_name": "John Doe",
            "confidence": 0.95
        }
    ]
    */
    
    -- Word-level Timestamps
    words JSONB,
    /* Structure:
    [
        {
            "word": "Hello",
            "start": 0.0,
            "end": 0.4,
            "confidence": 0.98
        }
    ]
    */
    
    -- Quality Metrics
    confidence_score FLOAT, -- Overall confidence 0-1
    has_speakers BOOLEAN DEFAULT false,
    speaker_count INT DEFAULT 0,
    
    -- Processing Info
    processing_time INT, -- seconds
    model_used VARCHAR(50), -- e.g., 'whisper-1'
    
    -- Metadata
    metadata JSONB DEFAULT '{}'::jsonb,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_transcripts_recording ON transcripts(recording_id);
CREATE INDEX idx_transcripts_room ON transcripts(room_id);
CREATE INDEX idx_transcripts_language ON transcripts(language);
CREATE UNIQUE INDEX idx_transcripts_recording_unique ON transcripts(recording_id);
CREATE INDEX idx_transcripts_segments ON transcripts USING GIN (segments);

-- Full-text search on transcript
CREATE INDEX idx_transcripts_text_search ON transcripts USING GIN (to_tsvector('english', text));
```

### meeting_summaries

**Purpose**: AI-generated meeting summaries and analysis

```sql
CREATE TABLE meeting_summaries (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- References
    room_id UUID NOT NULL UNIQUE REFERENCES rooms(id) ON DELETE CASCADE,
    transcript_id UUID REFERENCES transcripts(id),
    
    -- Summary Content
    executive_summary TEXT NOT NULL,
    
    -- Structured Data
    key_points JSONB DEFAULT '[]'::jsonb, -- Array of strings
    decisions JSONB DEFAULT '[]'::jsonb,
    /* Structure:
    [
        {
            "decision": "Launch product next month",
            "made_by": "user_id",
            "context": "...",
            "timestamp": "2024-11-03T10:30:00Z"
        }
    ]
    */
    
    topics JSONB DEFAULT '[]'::jsonb, -- Array of strings
    open_questions JSONB DEFAULT '[]'::jsonb, -- Array of strings
    next_steps JSONB DEFAULT '[]'::jsonb, -- Array of strings
    
    -- Sentiment Analysis
    overall_sentiment FLOAT, -- -1 (negative) to 1 (positive)
    sentiment_breakdown JSONB,
    /* Structure:
    {
        "positive_moments": [...],
        "negative_moments": [...],
        "neutral_moments": [...]
    }
    */
    
    -- Metrics
    total_speaking_time INT, -- seconds
    participant_balance_score FLOAT, -- 0-1, how balanced participation was
    engagement_score FLOAT, -- 0-1
    
    -- Model Info
    model_used VARCHAR(50), -- e.g., 'gpt-4-turbo'
    processing_time INT, -- seconds
    
    -- Metadata
    metadata JSONB DEFAULT '{}'::jsonb,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_summaries_room ON meeting_summaries(room_id);
CREATE INDEX idx_summaries_transcript ON meeting_summaries(transcript_id);
CREATE INDEX idx_summaries_sentiment ON meeting_summaries(overall_sentiment);
```

### action_items

**Purpose**: Tasks extracted from meetings

```sql
CREATE TABLE action_items (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- References
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    summary_id UUID REFERENCES meeting_summaries(id),
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
    transcript_reference TEXT, -- Quote from transcript
    timestamp_in_meeting INT, -- seconds from start of meeting
    
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
CREATE INDEX idx_action_items_assigned ON action_items(assigned_to);
CREATE INDEX idx_action_items_status ON action_items(status);
CREATE INDEX idx_action_items_priority ON action_items(priority);
CREATE INDEX idx_action_items_due_date ON action_items(due_date) WHERE status != 'completed';
CREATE INDEX idx_action_items_type ON action_items(type);
CREATE INDEX idx_action_items_tags ON action_items USING GIN (tags);

-- Trigger
CREATE TRIGGER update_action_items_updated_at BEFORE UPDATE ON action_items
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

### participant_reports

**Purpose**: Personalized reports for each meeting participant

```sql
CREATE TABLE participant_reports (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- References
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    participant_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    summary_id UUID REFERENCES meeting_summaries(id),
    
    -- Report Content
    report_content TEXT NOT NULL, -- Markdown format
    
    -- Participation Metrics
    speaking_time INT, -- seconds
    speaking_percentage FLOAT, -- 0-100
    contribution_count INT,
    questions_asked INT,
    interruptions INT,
    
    -- Engagement
    engagement_score FLOAT, -- 0-1
    attention_score FLOAT, -- 0-1 (based on activity)
    
    -- Contributions
    key_contributions JSONB DEFAULT '[]'::jsonb,
    /* Structure:
    [
        {
            "timestamp": 120,
            "quote": "I suggest we...",
            "impact": "high"
        }
    ]
    */
    
    -- Tasks
    tasks_assigned_count INT DEFAULT 0,
    tasks_created_count INT DEFAULT 0,
    
    -- Detailed Metrics
    metrics JSONB DEFAULT '{}'::jsonb,
    /* Structure:
    {
        "words_spoken": 1234,
        "avg_speaking_duration": 15.5,
        "longest_contribution": 120,
        "speaking_turns": 45
    }
    */
    
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
CREATE INDEX idx_reports_engagement ON participant_reports(engagement_score);
```

## Supporting Tables

### room_invitations

```sql
CREATE TABLE room_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    inviter_id UUID NOT NULL REFERENCES users(id),
    invitee_id UUID REFERENCES users(id),
    invitee_email VARCHAR(255),
    token VARCHAR(255) UNIQUE NOT NULL,
    status VARCHAR(20) DEFAULT 'pending' CHECK (
        status IN ('pending', 'accepted', 'declined', 'expired', 'revoked')
    ),
    message TEXT,
    expires_at TIMESTAMP NOT NULL,
    responded_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_invitations_room ON room_invitations(room_id);
CREATE INDEX idx_invitations_token ON room_invitations(token);
CREATE INDEX idx_invitations_email ON room_invitations(invitee_email);
```

### notifications

```sql
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT,
    data JSONB DEFAULT '{}'::jsonb,
    is_read BOOLEAN DEFAULT false,
    read_at TIMESTAMP,
    action_url TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_notifications_user ON notifications(user_id);
CREATE INDEX idx_notifications_read ON notifications(user_id, is_read, created_at DESC);
CREATE INDEX idx_notifications_type ON notifications(type);
```

### sessions

Stores refresh tokens and user sessions.

```sql
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    device_info JSONB,
    ip_address INET,
    user_agent TEXT,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    revoked_at TIMESTAMP,
    last_used_at TIMESTAMP
);

CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_hash ON sessions(token_hash);
CREATE INDEX idx_sessions_expires ON sessions(expires_at) WHERE revoked_at IS NULL;
```

## Helper Functions

### update_updated_at_column

```sql
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```

## Views

### active_meetings_view

```sql
CREATE VIEW active_meetings_view AS
SELECT 
    r.id,
    r.name,
    r.host_id,
    u.name as host_name,
    r.type,
    r.current_participants,
    r.max_participants,
    r.started_at,
    EXTRACT(EPOCH FROM (NOW() - r.started_at))::INT as duration_seconds,
    r.settings->>'enable_recording' as is_recording
FROM rooms r
JOIN users u ON r.host_id = u.id
WHERE r.status = 'active';
```

### user_statistics_view

```sql
CREATE VIEW user_statistics_view AS
SELECT 
    u.id,
    u.name,
    COUNT(DISTINCT r.id) as total_meetings_hosted,
    COUNT(DISTINCT p.room_id) as total_meetings_attended,
    SUM(p.duration) as total_time_in_meetings,
    COUNT(DISTINCT ai.id) as total_action_items,
    COUNT(DISTINCT CASE WHEN ai.status = 'completed' THEN ai.id END) as completed_action_items
FROM users u
LEFT JOIN rooms r ON r.host_id = u.id
LEFT JOIN participants p ON p.user_id = u.id
LEFT JOIN action_items ai ON ai.assigned_to = u.id
GROUP BY u.id, u.name;
```

## Data Retention Policies

```sql
-- Delete old recordings after 90 days
CREATE OR REPLACE FUNCTION cleanup_old_recordings()
RETURNS void AS $$
BEGIN
    DELETE FROM recordings
    WHERE created_at < NOW() - INTERVAL '90 days'
    AND status = 'completed';
END;
$$ LANGUAGE plpgsql;

-- Delete expired invitation tokens
CREATE OR REPLACE FUNCTION cleanup_expired_invitations()
RETURNS void AS $$
BEGIN
    UPDATE room_invitations
    SET status = 'expired'
    WHERE status = 'pending'
    AND expires_at < NOW();
END;
$$ LANGUAGE plpgsql;

-- Delete old notifications after 30 days
CREATE OR REPLACE FUNCTION cleanup_old_notifications()
RETURNS void AS $$
BEGIN
    DELETE FROM notifications
    WHERE created_at < NOW() - INTERVAL '30 days'
    AND is_read = true;
END;
$$ LANGUAGE plpgsql;
```

## Backup Strategy

- **Full backup**: Daily at 2 AM UTC
- **Incremental backup**: Every 6 hours
- **Point-in-time recovery**: Enabled (7 days)
- **Critical tables**: Hourly snapshots
- **Retention**: 30 days

## Redis Schema

```yaml
# User sessions
session:{user_id}:
  type: hash
  ttl: 7 days
  fields:
    user_id: string
    name: string
    email: string
    role: string
    last_active: timestamp

# Active room state
room:{room_id}:state:
  type: hash
  ttl: 24 hours
  fields:
    status: active|ended
    participant_count: number
    host_id: string
    started_at: timestamp

# Active participants in room
room:{room_id}:participants:
  type: set
  ttl: 24 hours
  members: [user_id1, user_id2, ...]

# Token blacklist
blacklist:token:{jti}:
  type: string
  ttl: token expiry time
  value: user_id

# Rate limiting
ratelimit:api:{user_id}:
  type: string
  ttl: 1 hour
  value: request_count

# OAuth state
oauth:state:{state}:
  type: hash
  ttl: 10 minutes
  fields:
    user_id: string
    provider: string
    timestamp: timestamp

# Processing queue
queue:ai:processing:
  type: list
  values: [recording_id1, recording_id2, ...]
```

## Monitoring Queries

```sql
-- Active meetings count
SELECT COUNT(*) FROM rooms WHERE status = 'active';

-- Current participants
SELECT SUM(current_participants) FROM rooms WHERE status = 'active';

-- Processing recordings
SELECT COUNT(*) FROM recordings WHERE status = 'processing';

-- Pending action items
SELECT COUNT(*) FROM action_items WHERE status = 'pending' AND due_date <= CURRENT_DATE + INTERVAL '7 days';

-- Storage usage
SELECT 
    pg_size_pretty(pg_database_size(current_database())) as database_size,
    pg_size_pretty(SUM(file_size)) as total_recording_size
FROM recordings;
```
