-- Drop triggers
DROP TRIGGER IF EXISTS trigger_calculate_room_duration ON rooms;
DROP TRIGGER IF EXISTS update_rooms_updated_at ON rooms;

-- Drop function
DROP FUNCTION IF EXISTS calculate_room_duration();

-- Drop indexes
DROP INDEX IF EXISTS idx_rooms_created;
DROP INDEX IF EXISTS idx_rooms_tags;
DROP INDEX IF EXISTS idx_rooms_livekit;
DROP INDEX IF EXISTS idx_rooms_slug;
DROP INDEX IF EXISTS idx_rooms_scheduled;
DROP INDEX IF EXISTS idx_rooms_type;
DROP INDEX IF EXISTS idx_rooms_status;
DROP INDEX IF EXISTS idx_rooms_host;

-- Drop table
DROP TABLE IF EXISTS rooms;
