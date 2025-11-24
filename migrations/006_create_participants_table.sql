-- +migrate Up

-- Create participants table
CREATE TABLE IF NOT EXISTS participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) DEFAULT 'participant' CHECK (role IN ('host', 'co_host', 'participant', 'guest')),
    status VARCHAR(20) DEFAULT 'invited' CHECK (status IN ('invited', 'joined', 'left', 'removed', 'declined')),
    invited_at TIMESTAMP,
    joined_at TIMESTAMP,
    left_at TIMESTAMP,
    duration INT,
    can_share_screen BOOLEAN DEFAULT true,
    can_record BOOLEAN DEFAULT false,
    can_mute_others BOOLEAN DEFAULT false,
    is_muted BOOLEAN DEFAULT false,
    is_hand_raised BOOLEAN DEFAULT false,
    is_removed BOOLEAN DEFAULT false,
    removed_by UUID REFERENCES users(id),
    removal_reason TEXT,
    connection_quality VARCHAR(20),
    device_info JSONB,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT unique_room_user UNIQUE (room_id, user_id)
);

-- Indexes for participants table
CREATE INDEX IF NOT EXISTS idx_participants_room ON participants(room_id);
CREATE INDEX IF NOT EXISTS idx_participants_user ON participants(user_id);
CREATE INDEX IF NOT EXISTS idx_participants_status ON participants(status);
CREATE INDEX IF NOT EXISTS idx_participants_joined ON participants(joined_at) WHERE joined_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_participants_role ON participants(role);
CREATE INDEX IF NOT EXISTS idx_participants_room_status ON participants(room_id, status);

-- +migrate Down
-- Rollback participants table and related objects
-- Drop indexes
DROP INDEX IF EXISTS idx_participants_room_status;
DROP INDEX IF EXISTS idx_participants_role;
DROP INDEX IF EXISTS idx_participants_joined;
DROP INDEX IF EXISTS idx_participants_status;
DROP INDEX IF EXISTS idx_participants_user;
DROP INDEX IF EXISTS idx_participants_room;

-- Drop table
DROP TABLE IF EXISTS participants;
