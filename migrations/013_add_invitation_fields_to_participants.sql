-- +migrate Up
-- Add invitation fields to participants table
ALTER TABLE participants
ADD COLUMN IF NOT EXISTS invited_email VARCHAR(255),
ADD COLUMN IF NOT EXISTS invited_by UUID REFERENCES users(id) ON DELETE SET NULL,
ADD COLUMN IF NOT EXISTS invited_at TIMESTAMP;

-- Add index for invited_email to query invitations quickly
CREATE INDEX IF NOT EXISTS idx_participants_invited_email ON participants(invited_email) 
WHERE invited_email IS NOT NULL;

-- Add index for invited_by
CREATE INDEX IF NOT EXISTS idx_participants_invited_by ON participants(invited_by)
WHERE invited_by IS NOT NULL;

COMMENT ON COLUMN participants.invited_email IS 'Email of invited user (before they register)';
COMMENT ON COLUMN participants.invited_by IS 'User who sent the invitation';
COMMENT ON COLUMN participants.invited_at IS 'When invitation was sent';

-- +migrate Down
DROP INDEX IF EXISTS idx_participants_invited_by;
DROP INDEX IF EXISTS idx_participants_invited_email;
ALTER TABLE participants
DROP COLUMN IF EXISTS invited_at,
DROP COLUMN IF EXISTS invited_by,
DROP COLUMN IF EXISTS invited_email;
