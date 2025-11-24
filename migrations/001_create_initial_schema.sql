-- +migrate Up

-- ============================================================================
-- USERS TABLE
-- ============================================================================

CREATE TABLE users (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Authentication
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255),
    oauth_provider VARCHAR(50),
    oauth_id VARCHAR(255),
    oauth_refresh_token TEXT,
    
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
CREATE INDEX idx_users_oauth ON users(oauth_provider, oauth_id) WHERE oauth_provider IS NOT NULL;
CREATE INDEX idx_users_role ON users(role) WHERE is_active = true;
CREATE INDEX idx_users_created ON users(created_at DESC);

-- ============================================================================
-- ROOMS TABLE
-- ============================================================================

CREATE TABLE rooms (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Basic Info
    name VARCHAR(255) NOT NULL,
    description TEXT,
    slug VARCHAR(100) UNIQUE,
    
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
    duration INT,
    
    -- Metadata
    tags TEXT[],
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
CREATE INDEX idx_rooms_active ON rooms(status) WHERE status = 'active';

-- ============================================================================
-- PARTICIPANTS TABLE
-- ============================================================================

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
    duration INT,
    
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
    connection_quality VARCHAR(20),
    device_info JSONB,
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
CREATE INDEX idx_participants_room_status ON participants(room_id, status);

-- +migrate Down
-- Rollback initial schema

-- Drop tables (in reverse order of dependencies)
DROP TABLE IF EXISTS participants;
DROP TABLE IF EXISTS rooms;
DROP TABLE IF EXISTS users;
