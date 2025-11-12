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

-- Trigger for updated_at
CREATE TRIGGER update_participants_updated_at 
    BEFORE UPDATE ON participants
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Function to calculate participant duration
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
    FOR EACH ROW 
    EXECUTE FUNCTION calculate_participant_duration();
