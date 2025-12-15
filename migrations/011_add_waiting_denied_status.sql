-- +migrate Up
-- Add 'waiting' and 'denied' status to participants status check constraint

-- Drop the old constraint
ALTER TABLE participants DROP CONSTRAINT IF EXISTS participants_status_check;

-- Add new constraint with waiting and denied status
ALTER TABLE participants ADD CONSTRAINT participants_status_check 
    CHECK (status IN ('invited', 'waiting', 'joined', 'left', 'removed', 'declined', 'denied'));

-- +migrate Down

-- Revert to old constraint without waiting and denied
ALTER TABLE participants DROP CONSTRAINT IF EXISTS participants_status_check;
ALTER TABLE participants ADD CONSTRAINT participants_status_check 
    CHECK (status IN ('invited', 'joined', 'left', 'removed', 'declined'));
