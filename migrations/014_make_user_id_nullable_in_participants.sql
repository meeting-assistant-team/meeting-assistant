-- +migrate Up
-- Make user_id nullable to support invitations before user registration
ALTER TABLE participants 
ALTER COLUMN user_id DROP NOT NULL;

-- Drop the unique constraint first (this will also drop the underlying index)
ALTER TABLE participants DROP CONSTRAINT IF EXISTS unique_room_user;

-- Add new constraint: unique on (room_id, user_id) when user_id is NOT NULL
CREATE UNIQUE INDEX unique_room_user_registered 
ON participants(room_id, user_id) 
WHERE user_id IS NOT NULL;

-- Add constraint: unique on (room_id, invited_email) when invited_email is NOT NULL
CREATE UNIQUE INDEX unique_room_invited_email
ON participants(room_id, invited_email)
WHERE invited_email IS NOT NULL;

-- Add check constraint: at least one of user_id or invited_email must be present
ALTER TABLE participants 
ADD CONSTRAINT check_user_or_email 
CHECK (user_id IS NOT NULL OR invited_email IS NOT NULL);

-- +migrate Down
-- Revert changes
ALTER TABLE participants DROP CONSTRAINT IF EXISTS check_user_or_email;
DROP INDEX IF EXISTS unique_room_invited_email;
DROP INDEX IF EXISTS unique_room_user_registered;

-- Recreate original unique constraint
ALTER TABLE participants 
ADD CONSTRAINT unique_room_user UNIQUE (room_id, user_id);

-- Make user_id NOT NULL again (this will fail if there are NULL values)
-- ALTER TABLE participants ALTER COLUMN user_id SET NOT NULL;
