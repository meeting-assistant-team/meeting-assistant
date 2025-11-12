-- Drop triggers
DROP TRIGGER IF EXISTS trigger_calculate_participant_duration ON participants;
DROP TRIGGER IF EXISTS update_participants_updated_at ON participants;

-- Drop function
DROP FUNCTION IF EXISTS calculate_participant_duration();

-- Drop indexes
DROP INDEX IF EXISTS idx_participants_room_status;
DROP INDEX IF EXISTS idx_participants_role;
DROP INDEX IF EXISTS idx_participants_joined;
DROP INDEX IF EXISTS idx_participants_status;
DROP INDEX IF EXISTS idx_participants_user;
DROP INDEX IF EXISTS idx_participants_room;

-- Drop table
DROP TABLE IF EXISTS participants;
