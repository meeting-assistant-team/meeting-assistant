-- +migrate Up
-- Migration: 005_create_views_and_maintenance
-- Created: 2024-11-05
-- Description: Create useful views and maintenance functions

-- ============================================================================
-- VIEWS
-- ============================================================================

-- Active meetings view
CREATE OR REPLACE VIEW active_meetings_view AS
SELECT 
    r.id,
    r.name,
    r.host_id,
    u.name as host_name,
    u.email as host_email,
    r.type,
    r.current_participants,
    r.max_participants,
    r.started_at,
    EXTRACT(EPOCH FROM (NOW() - r.started_at))::INT as duration_seconds,
    (r.settings->>'enable_recording')::boolean as is_recording_enabled,
    r.livekit_room_name
FROM rooms r
JOIN users u ON r.host_id = u.id
WHERE r.status = 'active';

-- User statistics view
CREATE OR REPLACE VIEW user_statistics_view AS
SELECT 
    u.id,
    u.name,
    u.email,
    COUNT(DISTINCT CASE WHEN r.host_id = u.id THEN r.id END) as total_meetings_hosted,
    COUNT(DISTINCT CASE WHEN p.user_id = u.id THEN p.room_id END) as total_meetings_attended,
    COALESCE(SUM(CASE WHEN p.user_id = u.id THEN p.duration END), 0) as total_time_in_meetings,
    COUNT(DISTINCT CASE WHEN ai.assigned_to = u.id THEN ai.id END) as total_action_items,
    COUNT(DISTINCT CASE WHEN ai.assigned_to = u.id AND ai.status = 'completed' THEN ai.id END) as completed_action_items,
    u.created_at as user_since
FROM users u
LEFT JOIN rooms r ON r.host_id = u.id
LEFT JOIN participants p ON p.user_id = u.id
LEFT JOIN action_items ai ON ai.assigned_to = u.id
WHERE u.is_active = true
GROUP BY u.id, u.name, u.email, u.created_at;

-- Room summary view
CREATE OR REPLACE VIEW room_summary_view AS
SELECT 
    r.id,
    r.name,
    r.type,
    r.status,
    r.host_id,
    u.name as host_name,
    r.scheduled_start_time,
    r.started_at,
    r.ended_at,
    r.duration,
    r.current_participants,
    r.max_participants,
    COUNT(DISTINCT p.id) as total_participants,
    COUNT(DISTINCT rec.id) as recording_count,
    EXISTS(SELECT 1 FROM meeting_summaries ms WHERE ms.room_id = r.id) as has_summary,
    COUNT(DISTINCT ai.id) as action_items_count,
    r.created_at
FROM rooms r
JOIN users u ON r.host_id = u.id
LEFT JOIN participants p ON p.room_id = r.id
LEFT JOIN recordings rec ON rec.room_id = r.id AND rec.status = 'completed'
LEFT JOIN action_items ai ON ai.room_id = r.id
GROUP BY r.id, r.name, r.type, r.status, r.host_id, u.name, r.scheduled_start_time, 
         r.started_at, r.ended_at, r.duration, r.current_participants, r.max_participants, r.created_at;

-- Pending action items view
CREATE OR REPLACE VIEW pending_action_items_view AS
SELECT 
    ai.id,
    ai.title,
    ai.description,
    ai.type,
    ai.priority,
    ai.status,
    ai.due_date,
    ai.assigned_to,
    u.name as assigned_to_name,
    u.email as assigned_to_email,
    ai.room_id,
    r.name as room_name,
    ai.created_at,
    CASE 
        WHEN ai.due_date < CURRENT_DATE THEN 'overdue'
        WHEN ai.due_date = CURRENT_DATE THEN 'due_today'
        WHEN ai.due_date <= CURRENT_DATE + INTERVAL '3 days' THEN 'due_soon'
        ELSE 'upcoming'
    END as urgency
FROM action_items ai
LEFT JOIN users u ON ai.assigned_to = u.id
LEFT JOIN rooms r ON ai.room_id = r.id
WHERE ai.status IN ('pending', 'in_progress')
ORDER BY 
    CASE ai.priority
        WHEN 'urgent' THEN 1
        WHEN 'high' THEN 2
        WHEN 'medium' THEN 3
        WHEN 'low' THEN 4
    END,
    ai.due_date NULLS LAST;

-- ============================================================================
-- MAINTENANCE FUNCTIONS
-- ============================================================================

-- Cleanup old recordings
CREATE OR REPLACE FUNCTION cleanup_old_recordings(retention_days INT DEFAULT 90)
RETURNS TABLE(deleted_count BIGINT) AS $$
DECLARE
    v_deleted_count BIGINT;
BEGIN
    WITH deleted AS (
        DELETE FROM recordings
        WHERE created_at < NOW() - (retention_days || ' days')::INTERVAL
        AND status IN ('completed', 'failed')
        RETURNING id
    )
    SELECT COUNT(*) INTO v_deleted_count FROM deleted;
    
    RETURN QUERY SELECT v_deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Cleanup expired invitations
CREATE OR REPLACE FUNCTION cleanup_expired_invitations()
RETURNS TABLE(updated_count BIGINT) AS $$
DECLARE
    v_updated_count BIGINT;
BEGIN
    WITH updated AS (
        UPDATE room_invitations
        SET status = 'expired'
        WHERE status = 'pending'
        AND expires_at < NOW()
        RETURNING id
    )
    SELECT COUNT(*) INTO v_updated_count FROM updated;
    
    RETURN QUERY SELECT v_updated_count;
END;
$$ LANGUAGE plpgsql;

-- Cleanup old notifications
CREATE OR REPLACE FUNCTION cleanup_old_notifications(retention_days INT DEFAULT 30)
RETURNS TABLE(deleted_count BIGINT) AS $$
DECLARE
    v_deleted_count BIGINT;
BEGIN
    WITH deleted AS (
        DELETE FROM notifications
        WHERE created_at < NOW() - (retention_days || ' days')::INTERVAL
        AND is_read = true
        RETURNING id
    )
    SELECT COUNT(*) INTO v_deleted_count FROM deleted;
    
    RETURN QUERY SELECT v_deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Cleanup expired sessions
CREATE OR REPLACE FUNCTION cleanup_expired_sessions()
RETURNS TABLE(deleted_count BIGINT) AS $$
DECLARE
    v_deleted_count BIGINT;
BEGIN
    WITH deleted AS (
        DELETE FROM sessions
        WHERE expires_at < NOW()
        OR revoked_at IS NOT NULL
        RETURNING id
    )
    SELECT COUNT(*) INTO v_deleted_count FROM deleted;
    
    RETURN QUERY SELECT v_deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Get database statistics
CREATE OR REPLACE FUNCTION get_database_statistics()
RETURNS TABLE(
    metric VARCHAR,
    value BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 'total_users'::VARCHAR, COUNT(*)::BIGINT FROM users WHERE is_active = true
    UNION ALL
    SELECT 'total_rooms'::VARCHAR, COUNT(*)::BIGINT FROM rooms
    UNION ALL
    SELECT 'active_rooms'::VARCHAR, COUNT(*)::BIGINT FROM rooms WHERE status = 'active'
    UNION ALL
    SELECT 'total_recordings'::VARCHAR, COUNT(*)::BIGINT FROM recordings
    UNION ALL
    SELECT 'completed_recordings'::VARCHAR, COUNT(*)::BIGINT FROM recordings WHERE status = 'completed'
    UNION ALL
    SELECT 'total_transcripts'::VARCHAR, COUNT(*)::BIGINT FROM transcripts
    UNION ALL
    SELECT 'total_summaries'::VARCHAR, COUNT(*)::BIGINT FROM meeting_summaries
    UNION ALL
    SELECT 'total_action_items'::VARCHAR, COUNT(*)::BIGINT FROM action_items
    UNION ALL
    SELECT 'pending_action_items'::VARCHAR, COUNT(*)::BIGINT FROM action_items WHERE status = 'pending'
    UNION ALL
    SELECT 'active_sessions'::VARCHAR, COUNT(*)::BIGINT FROM sessions WHERE expires_at > NOW() AND revoked_at IS NULL;
END;
$$ LANGUAGE plpgsql;

-- Get storage usage
CREATE OR REPLACE FUNCTION get_storage_usage()
RETURNS TABLE(
    category VARCHAR,
    size_bytes BIGINT,
    size_pretty TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        'database_total'::VARCHAR,
        pg_database_size(current_database())::BIGINT as size_bytes,
        pg_size_pretty(pg_database_size(current_database())) as size_pretty
    UNION ALL
    SELECT 
        'recordings_total'::VARCHAR,
        COALESCE(SUM(file_size), 0)::BIGINT,
        pg_size_pretty(COALESCE(SUM(file_size), 0)::BIGINT)
    FROM recordings
    WHERE status = 'completed';
END;
$$ LANGUAGE plpgsql;

-- +migrate Down
-- Rollback views and maintenance functions

-- Drop functions
DROP FUNCTION IF EXISTS get_storage_usage();
DROP FUNCTION IF EXISTS get_database_statistics();
DROP FUNCTION IF EXISTS cleanup_expired_sessions();
DROP FUNCTION IF EXISTS cleanup_old_notifications(INT);
DROP FUNCTION IF EXISTS cleanup_expired_invitations();
DROP FUNCTION IF EXISTS cleanup_old_recordings(INT);

-- Drop views
DROP VIEW IF EXISTS pending_action_items_view;
DROP VIEW IF EXISTS room_summary_view;
DROP VIEW IF EXISTS user_statistics_view;
DROP VIEW IF EXISTS active_meetings_view;
