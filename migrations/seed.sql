-- Seed data for development and testing
-- Run after migrations: psql -U postgres -d meeting_assistant -f migrations/seed.sql

BEGIN;

-- ============================================================================
-- SEED USERS
-- ============================================================================

-- Admin user
INSERT INTO users (id, email, password_hash, name, role, is_email_verified, avatar_url, created_at)
VALUES 
    ('11111111-1111-1111-1111-111111111111', 
     'admin@meetingassistant.com', 
     '$2a$10$YourHashedPasswordHere',  -- Replace with actual bcrypt hash
     'Admin User', 
     'admin', 
     true,
     'https://api.dicebear.com/7.x/avataaars/svg?seed=Admin',
     NOW() - INTERVAL '30 days')
ON CONFLICT (id) DO NOTHING;

-- Host users
INSERT INTO users (id, email, password_hash, name, role, is_email_verified, avatar_url, oauth_provider, oauth_id, created_at)
VALUES 
    ('22222222-2222-2222-2222-222222222222', 
     'john.doe@example.com', 
     '$2a$10$YourHashedPasswordHere',
     'John Doe', 
     'host', 
     true,
     'https://api.dicebear.com/7.x/avataaars/svg?seed=John',
     'google',
     'google_12345',
     NOW() - INTERVAL '20 days'),
    
    ('33333333-3333-3333-3333-333333333333', 
     'jane.smith@example.com', 
     NULL,  -- OAuth only
     'Jane Smith', 
     'host', 
     true,
     'https://api.dicebear.com/7.x/avataaars/svg?seed=Jane',
     'google',
     'google_67890',
     NOW() - INTERVAL '15 days')
ON CONFLICT (id) DO NOTHING;

-- Participant users
INSERT INTO users (id, email, password_hash, name, role, is_email_verified, avatar_url, created_at)
VALUES 
    ('44444444-4444-4444-4444-444444444444', 
     'alice.johnson@example.com', 
     '$2a$10$YourHashedPasswordHere',
     'Alice Johnson', 
     'participant', 
     true,
     'https://api.dicebear.com/7.x/avataaars/svg?seed=Alice',
     NOW() - INTERVAL '10 days'),
    
    ('55555555-5555-5555-5555-555555555555', 
     'bob.williams@example.com', 
     '$2a$10$YourHashedPasswordHere',
     'Bob Williams', 
     'participant', 
     true,
     'https://api.dicebear.com/7.x/avataaars/svg?seed=Bob',
     NOW() - INTERVAL '8 days'),
    
    ('66666666-6666-6666-6666-666666666666', 
     'carol.brown@example.com', 
     '$2a$10$YourHashedPasswordHere',
     'Carol Brown', 
     'participant', 
     true,
     'https://api.dicebear.com/7.x/avataaars/svg?seed=Carol',
     NOW() - INTERVAL '5 days')
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- SEED ROOMS
-- ============================================================================

-- Completed room with full data
INSERT INTO rooms (id, name, description, slug, host_id, type, status, livekit_room_name, max_participants, current_participants, scheduled_start_time, started_at, ended_at, tags, created_at)
VALUES 
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
     'Weekly Team Standup',
     'Our regular weekly team sync to discuss progress and blockers',
     'weekly-team-standup-001',
     '22222222-2222-2222-2222-222222222222',
     'scheduled',
     'ended',
     'lk_weekly_standup_001',
     10,
     0,
     NOW() - INTERVAL '3 days',
     NOW() - INTERVAL '3 days' + INTERVAL '5 minutes',
     NOW() - INTERVAL '3 days' + INTERVAL '45 minutes',
     ARRAY['standup', 'team', 'weekly'],
     NOW() - INTERVAL '7 days')
ON CONFLICT (id) DO NOTHING;

-- Active room
INSERT INTO rooms (id, name, description, slug, host_id, type, status, livekit_room_name, max_participants, current_participants, scheduled_start_time, started_at, tags, created_at)
VALUES 
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb',
     'Project Planning Session',
     'Planning session for Q4 initiatives',
     'project-planning-q4',
     '33333333-3333-3333-3333-333333333333',
     'scheduled',
     'active',
     'lk_project_planning_q4',
     15,
     4,
     NOW() - INTERVAL '30 minutes',
     NOW() - INTERVAL '25 minutes',
     ARRAY['planning', 'project', 'q4'],
     NOW() - INTERVAL '2 days')
ON CONFLICT (id) DO NOTHING;

-- Scheduled future room
INSERT INTO rooms (id, name, description, slug, host_id, type, status, livekit_room_name, max_participants, scheduled_start_time, scheduled_end_time, tags, created_at)
VALUES 
    ('cccccccc-cccc-cccc-cccc-cccccccccccc',
     'Client Demo',
     'Product demonstration for new client',
     'client-demo-acme',
     '22222222-2222-2222-2222-222222222222',
     'scheduled',
     'scheduled',
     'lk_client_demo_acme',
     20,
     NOW() + INTERVAL '2 days',
     NOW() + INTERVAL '2 days' + INTERVAL '1 hour',
     ARRAY['demo', 'client', 'sales'],
     NOW() - INTERVAL '1 day')
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- SEED PARTICIPANTS
-- ============================================================================

-- Participants for completed room
INSERT INTO participants (room_id, user_id, role, status, joined_at, left_at, duration)
VALUES 
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '22222222-2222-2222-2222-222222222222', 'host', 'left', NOW() - INTERVAL '3 days' + INTERVAL '5 minutes', NOW() - INTERVAL '3 days' + INTERVAL '45 minutes', 2400),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '44444444-4444-4444-4444-444444444444', 'participant', 'left', NOW() - INTERVAL '3 days' + INTERVAL '6 minutes', NOW() - INTERVAL '3 days' + INTERVAL '45 minutes', 2340),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '55555555-5555-5555-5555-555555555555', 'participant', 'left', NOW() - INTERVAL '3 days' + INTERVAL '7 minutes', NOW() - INTERVAL '3 days' + INTERVAL '45 minutes', 2280),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '66666666-6666-6666-6666-666666666666', 'participant', 'left', NOW() - INTERVAL '3 days' + INTERVAL '5 minutes', NOW() - INTERVAL '3 days' + INTERVAL '45 minutes', 2400)
ON CONFLICT (room_id, user_id) DO NOTHING;

-- Participants for active room
INSERT INTO participants (room_id, user_id, role, status, joined_at)
VALUES 
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '33333333-3333-3333-3333-333333333333', 'host', 'joined', NOW() - INTERVAL '25 minutes'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '22222222-2222-2222-2222-222222222222', 'co_host', 'joined', NOW() - INTERVAL '24 minutes'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '44444444-4444-4444-4444-444444444444', 'participant', 'joined', NOW() - INTERVAL '23 minutes'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '55555555-5555-5555-5555-555555555555', 'participant', 'joined', NOW() - INTERVAL '20 minutes')
ON CONFLICT (room_id, user_id) DO NOTHING;

-- Invited participants for scheduled room
INSERT INTO participants (room_id, user_id, role, status, invited_at)
VALUES 
    ('cccccccc-cccc-cccc-cccc-cccccccccccc', '22222222-2222-2222-2222-222222222222', 'host', 'invited', NOW() - INTERVAL '1 day'),
    ('cccccccc-cccc-cccc-cccc-cccccccccccc', '44444444-4444-4444-4444-444444444444', 'participant', 'invited', NOW() - INTERVAL '1 day'),
    ('cccccccc-cccc-cccc-cccc-cccccccccccc', '55555555-5555-5555-5555-555555555555', 'participant', 'invited', NOW() - INTERVAL '1 day')
ON CONFLICT (room_id, user_id) DO NOTHING;

-- ============================================================================
-- SEED RECORDINGS
-- ============================================================================

INSERT INTO recordings (id, room_id, started_by, livekit_recording_id, status, file_url, file_path, file_size, duration, started_at, completed_at, video_tracks, audio_tracks, resolution)
VALUES 
    ('dddddddd-dddd-dddd-dddd-dddddddddddd',
     'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
     '22222222-2222-2222-2222-222222222222',
     'lk_rec_001',
     'completed',
     'https://storage.example.com/recordings/weekly-standup-001.mp4',
     'recordings/2024/11/weekly-standup-001.mp4',
     524288000,  -- ~500MB
     2400,
     NOW() - INTERVAL '3 days' + INTERVAL '5 minutes',
     NOW() - INTERVAL '3 days' + INTERVAL '50 minutes',
     4,
     4,
     '1920x1080')
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- SEED TRANSCRIPTS
-- ============================================================================

INSERT INTO transcripts (id, recording_id, room_id, text, language, segments, confidence_score, has_speakers, speaker_count)
VALUES 
    ('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee',
     'dddddddd-dddd-dddd-dddd-dddddddddddd',
     'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
     'John: Good morning everyone. Let''s start our weekly standup. Alice, would you like to go first?

Alice: Sure! Last week I completed the user authentication module and started working on the room management features. No blockers currently.

Bob: I finished the LiveKit integration and it''s working well. I''m moving on to the recording functionality next.

Carol: I completed the database schema design and created all the migrations. Next, I''ll work on the API endpoints.',
     'en',
     '[
         {"id": 0, "start": 0.0, "end": 5.2, "text": "Good morning everyone. Let''s start our weekly standup.", "speaker": "22222222-2222-2222-2222-222222222222", "confidence": 0.95},
         {"id": 1, "start": 5.5, "end": 8.3, "text": "Alice, would you like to go first?", "speaker": "22222222-2222-2222-2222-222222222222", "confidence": 0.92},
         {"id": 2, "start": 8.8, "end": 15.2, "text": "Sure! Last week I completed the user authentication module and started working on the room management features.", "speaker": "44444444-4444-4444-4444-444444444444", "confidence": 0.94},
         {"id": 3, "start": 20.0, "end": 28.5, "text": "I finished the LiveKit integration and it''s working well. I''m moving on to the recording functionality next.", "speaker": "55555555-5555-5555-5555-555555555555", "confidence": 0.96}
     ]'::jsonb,
     0.94,
     true,
     4)
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- SEED MEETING SUMMARIES
-- ============================================================================

INSERT INTO meeting_summaries (id, room_id, transcript_id, executive_summary, key_points, decisions, topics, overall_sentiment, total_speaking_time, engagement_score)
VALUES 
    ('ffffffff-ffff-ffff-ffff-ffffffffffff',
     'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
     'eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee',
     'Weekly team standup meeting where team members shared progress updates. Authentication module was completed, LiveKit integration is functional, and database schema is ready. Team is making good progress on the meeting assistant project with no major blockers.',
     '["User authentication module completed", "LiveKit integration successful", "Database schema and migrations ready", "No blockers reported", "Team making good progress"]'::jsonb,
     '[{"decision": "Proceed with recording functionality implementation", "made_by": "22222222-2222-2222-2222-222222222222", "context": "LiveKit integration completed successfully"}]'::jsonb,
     '["Authentication", "LiveKit Integration", "Database Design", "API Development"]'::jsonb,
     0.85,
     2400,
     0.92)
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- SEED ACTION ITEMS
-- ============================================================================

INSERT INTO action_items (room_id, summary_id, assigned_to, created_by, title, description, type, priority, status, due_date, tags)
VALUES 
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
     'ffffffff-ffff-ffff-ffff-ffffffffffff',
     '44444444-4444-4444-4444-444444444444',
     '22222222-2222-2222-2222-222222222222',
     'Complete room management API endpoints',
     'Implement CRUD operations for rooms including create, update, delete, and list endpoints',
     'action',
     'high',
     'in_progress',
     CURRENT_DATE + INTERVAL '5 days',
     ARRAY['api', 'rooms', 'backend']),
    
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
     'ffffffff-ffff-ffff-ffff-ffffffffffff',
     '55555555-5555-5555-5555-555555555555',
     '22222222-2222-2222-2222-222222222222',
     'Implement recording functionality',
     'Build the recording feature using LiveKit egress API',
     'action',
     'high',
     'pending',
     CURRENT_DATE + INTERVAL '7 days',
     ARRAY['recording', 'livekit', 'backend']),
    
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
     'ffffffff-ffff-ffff-ffff-ffffffffffff',
     '66666666-6666-6666-6666-666666666666',
     '22222222-2222-2222-2222-222222222222',
     'Create API endpoint documentation',
     'Document all API endpoints with examples using Swagger',
     'action',
     'medium',
     'pending',
     CURRENT_DATE + INTERVAL '10 days',
     ARRAY['documentation', 'api', 'swagger'])
ON CONFLICT DO NOTHING;

-- ============================================================================
-- SEED NOTIFICATIONS
-- ============================================================================

INSERT INTO notifications (user_id, type, title, message, data, is_read, action_url)
VALUES 
    ('44444444-4444-4444-4444-444444444444',
     'action_item_assigned',
     'New task assigned',
     'You have been assigned: Complete room management API endpoints',
     '{"room_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "action_item_id": "abc123"}'::jsonb,
     false,
     '/action-items/abc123'),
    
    ('55555555-5555-5555-5555-555555555555',
     'meeting_invitation',
     'Meeting invitation',
     'You are invited to: Client Demo',
     '{"room_id": "cccccccc-cccc-cccc-cccc-cccccccccccc"}'::jsonb,
     false,
     '/rooms/cccccccc-cccc-cccc-cccc-cccccccccccc'),
    
    ('66666666-6666-6666-6666-666666666666',
     'meeting_summary_ready',
     'Meeting summary available',
     'Summary for Weekly Team Standup is now available',
     '{"room_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "summary_id": "ffffffff-ffff-ffff-ffff-ffffffffffff"}'::jsonb,
     true,
     '/rooms/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/summary')
ON CONFLICT DO NOTHING;

COMMIT;

-- Display summary
SELECT 'Seed data inserted successfully!' as message;
SELECT 'Users: ' || COUNT(*) as count FROM users;
SELECT 'Rooms: ' || COUNT(*) as count FROM rooms;
SELECT 'Participants: ' || COUNT(*) as count FROM participants;
SELECT 'Recordings: ' || COUNT(*) as count FROM recordings;
SELECT 'Transcripts: ' || COUNT(*) as count FROM transcripts;
SELECT 'Summaries: ' || COUNT(*) as count FROM meeting_summaries;
SELECT 'Action Items: ' || COUNT(*) as count FROM action_items;
SELECT 'Notifications: ' || COUNT(*) as count FROM notifications;
