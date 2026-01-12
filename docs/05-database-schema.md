# Database Schema

## Overview

PostgreSQL stores all persistent data. Redis handles sessions and caching.

## Core Entities

### users
**Purpose:** User accounts, authentication, and profile management

**Key Fields:**
- `id` (UUID, PK) - Primary identifier
- `email` (VARCHAR, UNIQUE) - User email (unique, case-insensitive index)
- `password_hash` (VARCHAR) - Hashed password for local auth
- `oauth_provider`, `oauth_id`, `oauth_refresh_token` - OAuth integration
- `name`, `avatar_url`, `bio` - Profile information
- `timezone`, `language` - Localization preferences
- `role` (VARCHAR) - admin, host, participant
- `is_active`, `is_email_verified` (BOOLEAN) - Account status
- `notification_preferences` (JSONB) - Email, push, reports settings
- `meeting_preferences` (JSONB) - Auto-join audio/video settings
- `last_login_at`, `last_active_at` - Activity tracking

**Indexes:**
- UNIQUE: `LOWER(email)`
- `(oauth_provider, oauth_id)` for OAuth users
- `role` for active users
- `created_at DESC`

---

### rooms
**Purpose:** Meeting room metadata and settings

**Key Fields:**
- `id` (UUID, PK)
- `name`, `description`, `slug` - Room identification
- `host_id` (UUID, FK → users) - Room owner
- `type` (VARCHAR) - public, private, scheduled
- `status` (VARCHAR) - scheduled, active, ended, cancelled
- `livekit_room_name`, `livekit_room_id` - LiveKit integration
- `max_participants` (INT, 2-100) - Room capacity
- `current_participants` (INT) - Active participant count
- `settings` (JSONB) - Recording, chat, screen share, waiting room, auto-record, transcription
- `scheduled_start_time`, `scheduled_end_time` - Scheduling
- `started_at`, `ended_at`, `duration` - Actual timing
- `tags` (JSONB) - Categorization
- `metadata` (JSONB) - Additional data

**Indexes:**
- `host_id`
- `status`, `type`
- `scheduled_start_time` (for scheduled rooms)
- `slug` (partial index)
- `livekit_room_name`
- GIN on `tags`

---

### participants
**Purpose:** Track user participation in rooms, including invitations

**Key Fields:**
- `id` (UUID, PK)
- `room_id` (UUID, FK → rooms)
- `user_id` (UUID, FK → users, nullable) - NULL for pending invitations
- `invited_email` (VARCHAR) - Email for invited users before registration
- `invited_by` (UUID, FK → users) - Who sent the invitation
- `role` (VARCHAR) - host, co_host, participant, guest
- `status` (VARCHAR) - invited, joined, left, removed, declined
- `invited_at`, `joined_at`, `left_at` - Timing
- `duration` (INT) - Participation duration
- `can_share_screen`, `can_record`, `can_mute_others` (BOOLEAN) - Permissions
- `is_muted`, `is_hand_raised` (BOOLEAN) - Current state
- `is_removed`, `removed_by`, `removal_reason` - Removal tracking
- `connection_quality` (VARCHAR)
- `device_info`, `metadata` (JSONB)

**Constraints:**
- UNIQUE `(room_id, user_id)` WHERE `user_id` IS NOT NULL
- UNIQUE `(room_id, invited_email)` WHERE `invited_email` IS NOT NULL
- CHECK: At least one of `user_id` or `invited_email` must be present

**Indexes:**
- `room_id`, `user_id`
- `status`, `role`
- `joined_at` (partial index)
- `(room_id, status)`
- `invited_email`, `invited_by` (partial indexes)

---

### recordings
**Purpose:** Track LiveKit recording sessions and files

**Key Fields:**
- `id` (UUID, PK)
- `room_id` (UUID, FK → rooms)
- `started_by` (UUID, FK → users) - Who initiated recording
- `livekit_recording_id`, `livekit_egress_id` (VARCHAR) - LiveKit references
- `status` (VARCHAR) - recording, processing, completed, failed, deleted
- `file_url`, `file_path` - Storage location
- `file_size` (BIGINT), `file_format` (VARCHAR) - File metadata
- `duration` (INT) - Recording duration in seconds
- `started_at`, `completed_at` - Timing
- `processing_started_at`, `processing_completed_at`, `processing_error` - Processing tracking
- `video_tracks`, `audio_tracks` (INT) - Track counts
- `resolution`, `bitrate` - Quality metrics
- `metadata` (JSONB)

**Indexes:**
- `room_id`
- `status`
- `livekit_recording_id` (partial index)
- `started_at DESC`
- `started_by` (partial index)

---

### transcripts
**Purpose:** Store AssemblyAI transcriptions with speaker diarization

**Key Fields:**
- `id` (UUID, PK)
- `meeting_id` (UUID, FK → rooms) - Main reference
- `recording_id`, `room_id` (VARCHAR) - Legacy backward compatibility
- `text` (TEXT) - Full transcript text
- `language` (VARCHAR) - Detected language
- `segments` (JSONB) - Detailed text segments
- `words` (JSONB) - Word-level timestamps
- `summary` (TEXT) - AI-generated summary (added in migration 010)
- `chapters` (JSONB) - Chapter markers (added in migration 010)
- `confidence_score` (FLOAT) - Overall confidence
- `has_speakers`, `speaker_count` (BOOLEAN, INT) - Diarization info
- `processing_time` (INT) - Processing duration
- `model_used` (VARCHAR) - Default: 'assemblyai'
- `raw_data` (JSONB) - Full AssemblyAI response
- `metadata` (JSONB)

**Indexes:**
- `meeting_id`, `recording_id`, `room_id`
- `language`
- GIN on `to_tsvector('english', text)` for full-text search

---

### transcript_utterances
**Purpose:** Detailed speaker segments with timestamps (migration 010)

**Key Fields:**
- `id` (UUID, PK)
- `transcript_id` (UUID, FK → transcripts)
- `speaker` (VARCHAR) - Speaker identifier
- `text` (TEXT) - Utterance text
- `start_time`, `end_time` (FLOAT) - Timeline position
- `confidence` (FLOAT) - Confidence score

**Indexes:**
- `transcript_id`
- `speaker`
- `start_time`

---

### meeting_summaries
**Purpose:** AI-generated meeting analysis and insights

**Key Fields:**
- `id` (UUID, PK)
- `room_id` (UUID, FK → rooms, UNIQUE) - One summary per room
- `transcript_id` (UUID, FK → transcripts)
- `executive_summary` (TEXT) - High-level summary
- `key_points`, `decisions`, `topics`, `open_questions`, `next_steps` (JSONB) - Structured insights
- `overall_sentiment` (FLOAT) - Sentiment score
- `sentiment_breakdown` (JSONB) - Detailed sentiment analysis
- `total_speaking_time` (INT) - Total meeting duration
- `participant_balance_score`, `engagement_score` (FLOAT) - Participation metrics
- `model_used` (VARCHAR) - AI model identifier
- `processing_time` (INT) - Processing duration
- `metadata` (JSONB)

**Indexes:**
- `room_id` (unique)
- `transcript_id` (partial index)
- `overall_sentiment` (partial index)
- `created_at DESC`

---

### action_items
**Purpose:** Tasks and follow-ups extracted from meetings

**Key Fields:**
- `id` (UUID, PK)
- `room_id` (UUID, FK → rooms)
- `summary_id` (UUID, FK → meeting_summaries)
- `assigned_to` (UUID, FK → users, nullable)
- `created_by` (UUID, FK → users)
- `title` (VARCHAR 500), `description` (TEXT) - Task details
- `type` (VARCHAR) - action, decision, question, follow_up, research
- `priority` (VARCHAR) - low, medium, high, urgent
- `status` (VARCHAR) - pending, in_progress, completed, cancelled, blocked
- `due_date` (DATE), `estimated_hours` (FLOAT)
- `transcript_reference` (TEXT), `timestamp_in_meeting` (INT) - Context
- `clickup_task_id`, `clickup_url`, `external_task_url` - External integrations
- `completed_at`, `completed_by`, `completion_notes` - Completion tracking
- `tags` (TEXT[])
- `metadata` (JSONB)

**Indexes:**
- `room_id`, `summary_id`, `assigned_to`, `created_by`
- `status`, `priority`, `type`
- `due_date` (for non-completed tasks)
- GIN on `tags`
- `(assigned_to, status)` for pending/in-progress tasks

---

### participant_reports
**Purpose:** Individual participation metrics and contributions

**Key Fields:**
- `id` (UUID, PK)
- `room_id` (UUID, FK → rooms)
- `participant_id` (UUID, FK → users)
- `summary_id` (UUID, FK → meeting_summaries)
- `report_content` (TEXT) - Textual report
- `speaking_time` (INT), `speaking_percentage` (FLOAT) - Speaking metrics
- `contribution_count`, `questions_asked`, `interruptions` (INT) - Participation counts
- `engagement_score`, `attention_score` (FLOAT) - Engagement metrics
- `key_contributions` (JSONB) - Highlighted contributions
- `tasks_assigned_count`, `tasks_created_count` (INT) - Task metrics
- `metrics`, `metadata` (JSONB)

**Constraints:**
- UNIQUE `(room_id, participant_id)`

**Indexes:**
- `room_id`, `participant_id`, `summary_id`
- `engagement_score` (partial index)

---

### ai_jobs
**Purpose:** Track asynchronous AI processing jobs (migration 007)

**Key Fields:**
- `id` (UUID, PK)
- `meeting_id` (UUID, FK → rooms)
- `job_type` (VARCHAR) - transcription, analysis, report_gen
- `status` (VARCHAR) - pending, submitted, processing, completed, failed, retrying, cancelled
- `external_job_id` (VARCHAR, UNIQUE) - AssemblyAI job ID
- `recording_url` (TEXT) - Source recording URL
- `transcript_id` (UUID, FK → transcripts)
- `started_at`, `completed_at` - Timing
- `retry_count`, `max_retries` (INT) - Retry management
- `last_error` (TEXT) - Error tracking
- `metadata` (JSONB)

**Indexes:**
- `meeting_id`
- `status`, `job_type`
- `external_job_id`
- `transcript_id`
- `(job_type, status)`
- `created_at`

---

## Supporting Tables

### sessions
**Purpose:** User session management

**Key Fields:**
- `id` (UUID, PK)
- `user_id` (UUID, FK → users)
- `token_hash` (VARCHAR, UNIQUE) - Hashed session token
- `device_info` (JSONB), `ip_address` (INET), `user_agent` (TEXT)
- `expires_at` (TIMESTAMP)
- `revoked_at`, `last_used_at` (TIMESTAMP)

**Indexes:**
- `user_id`
- `token_hash`
- `expires_at` (for active sessions)
- `(user_id, expires_at)` (for active sessions)

---

### token_families
**Purpose:** OAuth2 refresh token rotation tracking (migration 017)

**Key Fields:**
- `id` (UUID, PK)
- `user_id` (UUID, FK → users)
- `family_id` (UUID) - Token rotation chain identifier
- `refresh_token_hash` (VARCHAR, UNIQUE) - SHA256 hash of current token
- `parent_token_hash` (VARCHAR) - Previous token in chain
- `expires_at`, `revoked_at`, `last_used_at` (TIMESTAMP)
- `device_info` (JSONB), `ip_address` (INET), `user_agent` (TEXT)

**Indexes:**
- `user_id`
- `family_id`
- `expires_at`
- `refresh_token_hash`

**Purpose:** Detects token theft through rotation chain tracking

---

### room_invitations
**Purpose:** Room invitation management

**Key Fields:**
- `id` (UUID, PK)
- `room_id` (UUID, FK → rooms)
- `inviter_id` (UUID, FK → users)
- `invitee_id` (UUID, FK → users, nullable)
- `invitee_email` (VARCHAR) - For guest invitations
- `token` (VARCHAR, UNIQUE) - Invitation token
- `status` (VARCHAR) - pending, accepted, declined, expired, revoked
- `message` (TEXT) - Invitation message
- `expires_at`, `responded_at` (TIMESTAMP)

**Constraints:**
- CHECK: Either `invitee_id` OR `invitee_email` must be present

**Indexes:**
- `room_id`, `inviter_id`, `invitee_id`
- `token`, `invitee_email`
- `status`
- `(status, expires_at)` for pending invitations

---

### notifications
**Purpose:** User notifications and alerts

**Key Fields:**
- `id` (UUID, PK)
- `user_id` (UUID, FK → users)
- `type` (VARCHAR) - Notification type
- `title` (VARCHAR), `message` (TEXT)
- `data` (JSONB) - Additional notification data
- `is_read` (BOOLEAN), `read_at` (TIMESTAMP)
- `action_url` (TEXT) - CTA link

**Indexes:**
- `user_id`
- `(user_id, is_read, created_at DESC)`
- `type`
- `(user_id, created_at DESC)` for unread notifications

---

## Relationships

```
users (1) ──────────> (*) rooms [host_id]
users (1) ──────────> (*) participants [user_id]
users (1) ──────────> (*) participants [invited_by]
users (1) ──────────> (*) action_items [assigned_to]
users (1) ──────────> (*) action_items [created_by]
users (1) ──────────> (*) sessions
users (1) ──────────> (*) token_families
users (1) ──────────> (*) notifications
users (1) ──────────> (*) room_invitations [inviter_id]
users (1) ──────────> (*) room_invitations [invitee_id]

rooms (1) ──────────> (*) participants
rooms (1) ──────────> (*) recordings
rooms (1) ──────────> (*) transcripts [meeting_id]
rooms (1) ──────────> (1) meeting_summaries
rooms (1) ──────────> (*) action_items
rooms (1) ──────────> (*) participant_reports
rooms (1) ──────────> (*) ai_jobs [meeting_id]
rooms (1) ──────────> (*) room_invitations

transcripts (1) ────> (*) transcript_utterances
transcripts (1) ────> (1) meeting_summaries
transcripts (1) ────> (*) ai_jobs

meeting_summaries (1) ─> (*) action_items
meeting_summaries (1) ─> (*) participant_reports
```

---

## Key Features

### 1. **Flexible Invitation System**
- Participants can be invited before registration (`invited_email`)
- Tracks invitation status and inviter
- Supports both registered users and email-based invitations

### 2. **OAuth2 Token Security**
- Token families track refresh token rotation
- Detects token theft through parent-child chain
- Supports automatic token revocation

### 3. **AI Processing Pipeline**
- `ai_jobs` tracks asynchronous processing
- Retry mechanism for failed jobs
- External job ID for AssemblyAI integration

### 4. **Speaker Diarization**
- `transcript_utterances` stores detailed speaker segments
- Timeline-based queries with start/end times
- Speaker-specific analysis support

### 5. **Full-Text Search**
- GIN index on transcript text
- Tag-based filtering on rooms and action items
- Efficient text search capabilities

---

## Data Retention

- **Active meetings:** Indefinitely
- **Old recordings:** 90 days (configurable)
- **Transcripts:** Indefinitely
- **Notifications:** 30 days
- **Sessions:** Until expiration/revocation
- **Token families:** Until expiration/revocation
- **Backups:** 30-day retention

---

## Migration History

1. `001` - Initial schema (users, rooms, participants)
2. `002` - Recordings and transcripts
3. `003` - AI summaries, action items, participant reports
4. `004` - Rooms table (duplicate/update)
5. `005` - Supporting tables (sessions, invitations, notifications)
6. `006` - Participants table (duplicate/update)
7. `007` - AI jobs table
8. `008` - Update transcripts schema
9. `009` - Fix external_job_id unique constraint
10. `010` - Transcript utterances and summary/chapters fields
11. `011` - Add waiting/denied status
12. `012` - Remove external_job_id unique constraint
13. `013` - Add invitation fields to participants
14. `014` - Make user_id nullable in participants
15. `015` - Remove unique invited_email constraint
16. `016` - Add AI job statuses
17. `017` - Token families for OAuth2 token rotation
