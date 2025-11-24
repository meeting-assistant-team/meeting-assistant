-- +migrate Up
-- Migration: 004_create_supporting_tables
-- Created: 2024-11-05
-- Description: Create supporting tables for sessions, invitations, and notifications

-- ============================================================================
-- SESSIONS TABLE
-- ============================================================================

CREATE TABLE sessions (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- References
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Token
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    
    -- Device Info
    device_info JSONB,
    ip_address INET,
    user_agent TEXT,
    
    -- Lifecycle
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    revoked_at TIMESTAMP,
    last_used_at TIMESTAMP
);

-- Indexes
CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_hash ON sessions(token_hash);
CREATE INDEX idx_sessions_expires ON sessions(expires_at) WHERE revoked_at IS NULL;
CREATE INDEX idx_sessions_user_active ON sessions(user_id, expires_at) 
    WHERE revoked_at IS NULL;

-- ============================================================================
-- ROOM_INVITATIONS TABLE
-- ============================================================================

CREATE TABLE room_invitations (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- References
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    inviter_id UUID NOT NULL REFERENCES users(id),
    invitee_id UUID REFERENCES users(id),
    
    -- Invitee Info (for guests)
    invitee_email VARCHAR(255),
    
    -- Token
    token VARCHAR(255) UNIQUE NOT NULL,
    
    -- Status
    status VARCHAR(20) DEFAULT 'pending' CHECK (
        status IN ('pending', 'accepted', 'declined', 'expired', 'revoked')
    ),
    
    -- Message
    message TEXT,
    
    -- Lifecycle
    expires_at TIMESTAMP NOT NULL,
    responded_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT invitee_required CHECK (
        invitee_id IS NOT NULL OR invitee_email IS NOT NULL
    )
);

-- Indexes
CREATE INDEX idx_invitations_room ON room_invitations(room_id);
CREATE INDEX idx_invitations_inviter ON room_invitations(inviter_id);
CREATE INDEX idx_invitations_invitee ON room_invitations(invitee_id) WHERE invitee_id IS NOT NULL;
CREATE INDEX idx_invitations_token ON room_invitations(token);
CREATE INDEX idx_invitations_email ON room_invitations(invitee_email) WHERE invitee_email IS NOT NULL;
CREATE INDEX idx_invitations_status ON room_invitations(status);
CREATE INDEX idx_invitations_pending ON room_invitations(status, expires_at) 
    WHERE status = 'pending';

-- ============================================================================
-- NOTIFICATIONS TABLE
-- ============================================================================

CREATE TABLE notifications (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- References
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Notification Details
    type VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT,
    
    -- Data
    data JSONB DEFAULT '{}'::jsonb,
    
    -- Read Status
    is_read BOOLEAN DEFAULT false,
    read_at TIMESTAMP,
    
    -- Action
    action_url TEXT,
    
    -- Metadata
    created_at TIMESTAMP DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_notifications_user ON notifications(user_id);
CREATE INDEX idx_notifications_read ON notifications(user_id, is_read, created_at DESC);
CREATE INDEX idx_notifications_type ON notifications(type);
CREATE INDEX idx_notifications_user_unread ON notifications(user_id, created_at DESC) 
    WHERE is_read = false;

-- +migrate Down
-- Rollback supporting tables

-- Drop tables
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS room_invitations;
DROP TABLE IF EXISTS sessions;
