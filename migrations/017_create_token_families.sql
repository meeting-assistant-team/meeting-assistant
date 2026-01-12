-- +migrate Up
-- 017_create_token_families.sql
-- Token families table for refresh token rotation tracking

CREATE TABLE IF NOT EXISTS token_families (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    family_id UUID NOT NULL, -- Tracks token rotation chain (multiple tokens can share same family)
    refresh_token_hash VARCHAR(255) NOT NULL UNIQUE, -- SHA256 hash of current refresh token
    parent_token_hash VARCHAR(255), -- Hash of previous token (for rotation detection)
    created_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP,
    last_used_at TIMESTAMP,
    
    -- Device tracking
    device_info JSONB,
    ip_address INET,
    user_agent TEXT
);

-- Indexes
CREATE INDEX idx_token_families_user_id ON token_families(user_id);
CREATE INDEX idx_token_families_family_id ON token_families(family_id);
CREATE INDEX idx_token_families_expires_at ON token_families(expires_at);
CREATE INDEX idx_token_families_refresh_token_hash ON token_families(refresh_token_hash);

-- Add comment
COMMENT ON TABLE token_families IS 'Tracks refresh token rotation families for security (detects token theft)';
COMMENT ON COLUMN token_families.family_id IS 'Unique identifier for token rotation chain';
COMMENT ON COLUMN token_families.refresh_token_hash IS 'SHA256 hash of current refresh token';
COMMENT ON COLUMN token_families.parent_token_hash IS 'Hash of previous token in rotation chain';

-- +migrate Down
DROP TABLE IF EXISTS token_families;
