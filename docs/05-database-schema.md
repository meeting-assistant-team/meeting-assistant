# Database Schema

## Overview

PostgreSQL stores all persistent data. Redis handles sessions and caching.

## Core Entities

### users
- User accounts and profiles
- OAuth provider information
- Role (admin, host, participant)
- Notification preferences

### rooms
- Meeting room metadata
- Room status (scheduled, active, ended)
- Host information
- Recording settings

### participants
- Join records linking users to rooms
- Admission status (pending, approved, rejected, left)
- Join/leave timestamps
- Participant role in meeting

### recordings
- Raw recording files from LiveKit
- Storage location references
- Duration and quality metadata
- Processing status

### transcripts
- Full-text transcriptions from AssemblyAI
- Speaker diarization data
- Confidence scores
- Processing metadata

### meeting_summaries
- AI-generated executive summary
- Key discussion points
- Sentiment analysis results
- Generated at analysis time

### action_items
- Tasks extracted from meeting
- Assigned to participants
- Due dates and priority
- Completion status

### notifications
- Event notifications for users
- Read/unread status
- Delivery status

## Relationships

- **users** → many **rooms** (host relationship)
- **rooms** → many **participants** (members)
- **rooms** → many **recordings** (one per recording session)
- **recordings** → one **transcript** (after transcription)
- **transcripts** → one **meeting_summary** (after analysis)
- **meeting_summary** → many **action_items** (extracted tasks)
- **rooms** → many **notifications** (event notifications)

## Indexes

Key indexes for performance:
- `users.email` (UNIQUE)
- `participants.room_id, user_id`
- `recordings.room_id, created_at`
- `transcripts.recording_id`
- `action_items.assigned_to, due_date`

## Data Retention

- Active meetings: Indefinitely
- Old recordings: 90 days (configurable)
- Transcripts: Indefinitely
- Notifications: 30 days
- Backups: 30-day retention
