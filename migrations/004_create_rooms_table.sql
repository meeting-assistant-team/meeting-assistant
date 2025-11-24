-- +migrate Up

-- Create rooms table
CREATE TABLE IF NOT EXISTS rooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    slug VARCHAR(100) UNIQUE,
    host_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL DEFAULT 'public' CHECK (type IN ('public', 'private', 'scheduled')),
    status VARCHAR(20) NOT NULL DEFAULT 'scheduled' CHECK (status IN ('scheduled', 'active', 'ended', 'cancelled')),
    livekit_room_name VARCHAR(255) UNIQUE NOT NULL,
    livekit_room_id VARCHAR(255),
    max_participants INT DEFAULT 10 CHECK (max_participants >= 2 AND max_participants <= 100),
    current_participants INT DEFAULT 0,
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
    scheduled_start_time TIMESTAMP,
    scheduled_end_time TIMESTAMP,
    started_at TIMESTAMP,
    ended_at TIMESTAMP,
    duration INT,
    tags JSONB DEFAULT '[]'::jsonb,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for rooms table
CREATE INDEX IF NOT EXISTS idx_rooms_host ON rooms(host_id);
CREATE INDEX IF NOT EXISTS idx_rooms_status ON rooms(status);
CREATE INDEX IF NOT EXISTS idx_rooms_type ON rooms(type);
CREATE INDEX IF NOT EXISTS idx_rooms_scheduled ON rooms(scheduled_start_time) WHERE status = 'scheduled';
CREATE INDEX IF NOT EXISTS idx_rooms_slug ON rooms(slug) WHERE slug IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_rooms_livekit ON rooms(livekit_room_name);
CREATE INDEX IF NOT EXISTS idx_rooms_tags ON rooms USING GIN (tags);
CREATE INDEX IF NOT EXISTS idx_rooms_created ON rooms(created_at DESC);

-- +migrate Down
-- Rollback rooms table and related objects

-- Drop table
DROP TABLE IF EXISTS rooms;
