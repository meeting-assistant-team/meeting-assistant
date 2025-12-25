-- +migrate Up
-- Remove unique constraint on (room_id, invited_email) to allow re-invitations
-- Logic now handles duplicate invitations in application code

DROP INDEX IF EXISTS unique_room_invited_email;

-- +migrate Down
-- Restore unique constraint
CREATE UNIQUE INDEX unique_room_invited_email
ON participants(room_id, invited_email)
WHERE invited_email IS NOT NULL;
