# API Documentation

## Base URL

```
Development: http://localhost:8080/api/v1
Production: https://api.meetingassistant.com/api/v1
```

## Authentication

All API requests (except auth endpoints) require authentication via JWT Bearer token.

```
Authorization: Bearer {access_token}
```

## Response Format

### Success Response

```json
{
  "success": true,
  "data": {
    // Response data
  },
  "metadata": {
    "timestamp": "2025-11-03T10:00:00Z",
    "request_id": "uuid"
  }
}
```

### Error Response

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable error message",
    "details": "Additional error details",
    "field": "field_name" // for validation errors
  },
  "metadata": {
    "timestamp": "2025-11-03T10:00:00Z",
    "request_id": "uuid"
  }
}
```

## API Endpoints

### Authentication

#### GET /auth/google
Initiate Google OAuth2 flow. Redirects to Google consent page.

**Response:** `302 Redirect`
Redirects to Google OAuth consent page.

#### GET /auth/google/callback
Handle Google OAuth callback and complete authentication.

**Query Parameters:**
- `code`: Authorization code from Google
- `state`: CSRF protection token

**Response:** `200 OK`
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "uuid",
      "email": "user@example.com",
      "name": "John Doe",
      "avatar_url": "https://lh3.googleusercontent.com/...",
      "role": "participant"
    },
    "access_token": "jwt_access_token",
    "refresh_token": "jwt_refresh_token",
    "expires_in": 900,
    "is_new_user": false
  }
}
```

#### POST /auth/refresh
Refresh access token using refresh token.

**Request Body:**
```json
{
  "refresh_token": "refresh_token_here"
}
```

**Response:** `200 OK`
```json
{
  "success": true,
  "data": {
    "access_token": "new_jwt_access_token",
    "refresh_token": "new_refresh_token",
    "expires_in": 900
  }
}
```

#### POST /auth/logout
Logout and revoke session.

**Headers:**
```
Authorization: Bearer {access_token}
```

**Request Body:**
```json
{
  "refresh_token": "refresh_token_here"
}
```

**Response:** `200 OK`
```json
{
  "success": true,
  "data": {
    "message": "Logged out successfully"
  }
}
```

#### GET /auth/me
Get current authenticated user info.

**Headers:**
```
Authorization: Bearer {access_token}
```

**Response:** `200 OK`
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "email": "user@example.com",
    "name": "John Doe",
    "avatar_url": "https://lh3.googleusercontent.com/...",
    "role": "participant",
    "oauth_provider": "google",
    "created_at": "2025-01-01T00:00:00Z",
    "last_login_at": "2025-11-03T10:00:00Z"
  }
}
```

### Users

#### GET /users/:id
Get user profile

**Response:** `200 OK`
```json
{
  "id": "uuid",
  "name": "John Doe",
  "email": "user@example.com",
  "avatar_url": "https://...",
  "bio": "Product Manager",
  "role": "host",
  "created_at": "2025-01-01T00:00:00Z"
}
```

#### PATCH /users/:id
Update user profile

**Request Body:**
```json
{
  "name": "John Smith",
  "bio": "Updated bio",
  "avatar_url": "https://...",
  "notification_preferences": {
    "email": true,
    "push": true,
    "reports": true
  }
}
```

**Response:** `200 OK`

#### GET /users/:id/statistics
Get user statistics

**Response:** `200 OK`
```json
{
  "total_meetings_hosted": 15,
  "total_meetings_attended": 42,
  "total_time_in_meetings": 36000,
  "total_action_items": 38,
  "completed_action_items": 25,
  "completion_rate": 0.66
}
```

### Rooms

#### POST /rooms
Create a new room

**Request Body:**
```json
{
  "name": "Product Planning Meeting",
  "description": "Weekly product planning sync",
  "type": "private",
  "max_participants": 10,
  "scheduled_start_time": "2025-11-10T14:00:00Z",
  "settings": {
    "enable_recording": true,
    "enable_chat": true,
    "require_approval": true,
    "auto_record": true
  }
}
```

**Response:** `201 Created`
```json
{
  "room": {
    "id": "uuid",
    "name": "Product Planning Meeting",
    "slug": "product-planning-meeting-abc123",
    "type": "private",
    "status": "scheduled",
    "host_id": "uuid",
    "livekit_room_name": "room_xxx",
    "max_participants": 10,
    "current_participants": 0,
    "scheduled_start_time": "2025-11-10T14:00:00Z",
    "settings": {...},
    "created_at": "2025-11-03T10:00:00Z"
  },
  "livekit_token": "jwt_token_for_livekit",
  "livekit_url": "wss://livekit.example.com"
}
```

#### GET /rooms
List rooms

**Query Parameters:**
- `type`: `public` | `private` | `all` (default: `all`)
- `status`: `scheduled` | `active` | `ended` (default: `active`)
- `limit`: number (default: 20, max: 100)
- `offset`: number (default: 0)
- `search`: string (search in name and description)

**Response:** `200 OK`
```json
{
  "rooms": [
    {
      "id": "uuid",
      "name": "Team Standup",
      "type": "public",
      "status": "active",
      "host": {
        "id": "uuid",
        "name": "Jane Doe"
      },
      "current_participants": 5,
      "max_participants": 10,
      "started_at": "2025-11-03T09:00:00Z"
    }
  ],
  "total": 15,
  "limit": 20,
  "offset": 0
}
```

#### GET /rooms/:id
Get room details

**Response:** `200 OK`
```json
{
  "room": {
    "id": "uuid",
    "name": "Team Standup",
    "description": "Daily standup meeting",
    "type": "public",
    "status": "active",
    "host_id": "uuid",
    "max_participants": 10,
    "current_participants": 5,
    "settings": {...},
    "started_at": "2025-11-03T09:00:00Z",
    "created_at": "2025-11-03T08:55:00Z"
  },
  "participants": [
    {
      "id": "uuid",
      "user": {
        "id": "uuid",
        "name": "John Doe",
        "avatar_url": "https://..."
      },
      "role": "host",
      "joined_at": "2025-11-03T09:00:00Z"
    }
  ],
  "host": {
    "id": "uuid",
    "name": "Jane Doe",
    "avatar_url": "https://..."
  }
}
```

#### PATCH /rooms/:id
Update room details

**Request Body:**
```json
{
  "name": "Updated Room Name",
  "description": "Updated description",
  "settings": {
    "enable_recording": false
  }
}
```

**Response:** `200 OK`

#### DELETE /rooms/:id
Delete room

**Response:** `204 No Content`

#### GET /rooms/:id/join
Join a room

**Query Parameters:**
- `token`: Invitation token (required for private rooms)

**Response:** `200 OK`
```json
{
  "room": {...},
  "livekit_token": "jwt_token",
  "livekit_url": "wss://...",
  "participants": [...]
}
```

#### POST /rooms/:id/leave
Leave a room

**Response:** `200 OK`

#### POST /rooms/:id/end
End a meeting (host only)

**Response:** `200 OK`
```json
{
  "message": "Meeting ended successfully",
  "duration": 3600,
  "participants_count": 5
}
```

#### POST /rooms/:id/invite
Invite participants to room

**Request Body:**
```json
{
  "emails": ["user1@example.com", "user2@example.com"],
  "message": "Please join our meeting"
}
```

**Response:** `200 OK`
```json
{
  "sent": 2,
  "failed": 0,
  "invitations": [
    {
      "email": "user1@example.com",
      "token": "invite_token",
      "expires_at": "2025-11-10T14:00:00Z"
    }
  ]
}
```

### Participants

#### GET /rooms/:id/participants
List room participants

**Response:** `200 OK`
```json
{
  "participants": [
    {
      "id": "uuid",
      "user_id": "uuid",
      "name": "John Doe",
      "avatar_url": "https://...",
      "role": "host",
      "status": "joined",
      "joined_at": "2025-11-03T09:00:00Z",
      "is_muted": false,
      "is_hand_raised": false
    }
  ]
}
```

#### POST /rooms/:id/participants/:participantId/mute
Mute a participant (host only)

**Response:** `200 OK`

#### POST /rooms/:id/participants/:participantId/unmute
Unmute a participant

**Response:** `200 OK`

#### DELETE /rooms/:id/participants/:participantId
Remove participant from room (host only)

**Request Body:**
```json
{
  "reason": "Disruptive behavior"
}
```

**Response:** `200 OK`

#### POST /rooms/:id/transfer-host
Transfer host role to another participant

**Request Body:**
```json
{
  "new_host_id": "uuid"
}
```

**Response:** `200 OK`

### Recordings

#### GET /rooms/:id/recordings
List room recordings

**Response:** `200 OK`
```json
{
  "recordings": [
    {
      "id": "uuid",
      "room_id": "uuid",
      "status": "completed",
      "file_url": "https://...",
      "file_size": 125829120,
      "duration": 3600,
      "format": "mp4",
      "started_at": "2025-11-03T09:00:00Z",
      "completed_at": "2025-11-03T10:05:00Z"
    }
  ]
}
```

#### POST /rooms/:id/recording/start
Start recording (host only)

**Response:** `200 OK`
```json
{
  "recording_id": "uuid",
  "status": "recording",
  "started_at": "2025-11-03T09:00:00Z"
}
```

#### POST /rooms/:id/recording/stop
Stop recording

**Response:** `200 OK`
```json
{
  "recording_id": "uuid",
  "status": "processing",
  "duration": 3600
}
```

#### GET /recordings/:id
Get recording details

**Response:** `200 OK`

#### GET /recordings/:id/download
Download recording file

**Response:** `302 Redirect` to S3 signed URL

### Transcripts

#### GET /meetings/:id/transcript
Get meeting transcript

**Response:** `200 OK`
```json
{
  "transcript_id": "uuid",
  "meeting_id": "uuid",
  "text": "Full transcript text...",
  "language": "en",
  "segments": [
    {
      "id": 0,
      "start": 0.0,
      "end": 5.5,
      "text": "Hello everyone, let's get started.",
      "speaker": "uuid",
      "speaker_name": "John Doe",
      "confidence": 0.95
    }
  ],
  "has_speakers": true,
  "speaker_count": 5,
  "confidence_score": 0.93,
  "created_at": "2025-11-03T10:10:00Z"
}
```

#### GET /meetings/:id/transcript/search
Search in transcript

**Query Parameters:**
- `q`: Search query (required)
- `speaker`: Filter by speaker user_id (optional)

**Response:** `200 OK`
```json
{
  "results": [
    {
      "segment_id": 42,
      "start": 120.5,
      "end": 125.0,
      "text": "We should prioritize the mobile app",
      "speaker_name": "Jane Doe",
      "context_before": "...",
      "context_after": "..."
    }
  ],
  "total_matches": 3
}
```

### Summaries

#### GET /meetings/:id/summary
Get meeting summary

**Response:** `200 OK`
```json
{
  "summary_id": "uuid",
  "meeting_id": "uuid",
  "executive_summary": "The team discussed Q4 goals...",
  "key_points": [
    "Agreed to launch mobile app in Q1",
    "Budget approved for hiring 2 developers"
  ],
  "decisions": [
    {
      "decision": "Launch mobile app in Q1 2026",
      "made_by": "uuid",
      "context": "After reviewing market analysis..."
    }
  ],
  "topics": ["Mobile App", "Budget", "Hiring"],
  "open_questions": [
    "What should be the MVP features?"
  ],
  "next_steps": [
    "John to create project timeline",
    "Jane to interview candidates"
  ],
  "overall_sentiment": 0.7,
  "engagement_score": 0.85,
  "created_at": "2025-11-03T10:15:00Z"
}
```

### Action Items

#### GET /meetings/:id/action-items
Get meeting action items

**Query Parameters:**
- `assigned_to`: Filter by assignee user_id
- `status`: Filter by status
- `priority`: Filter by priority

**Response:** `200 OK`
```json
{
  "action_items": [
    {
      "id": "uuid",
      "meeting_id": "uuid",
      "title": "Create project timeline",
      "description": "Detailed timeline for mobile app development",
      "assigned_to": {
        "id": "uuid",
        "name": "John Doe",
        "email": "john@example.com"
      },
      "type": "action",
      "priority": "high",
      "status": "pending",
      "due_date": "2025-11-10",
      "transcript_reference": "John, can you create a timeline?",
      "clickup_url": "https://app.clickup.com/t/...",
      "created_at": "2025-11-03T10:15:00Z"
    }
  ],
  "total": 5
}
```

#### GET /action-items
Get all user's action items

**Query Parameters:**
- `status`: Filter by status
- `due_before`: ISO date string
- `limit`, `offset`: Pagination

**Response:** `200 OK`

#### PATCH /action-items/:id
Update action item

**Request Body:**
```json
{
  "status": "in_progress",
  "completion_notes": "Started working on it",
  "due_date": "2025-11-15"
}
```

**Response:** `200 OK`

#### POST /action-items
Create manual action item

**Request Body:**
```json
{
  "meeting_id": "uuid",
  "title": "Follow up with client",
  "description": "Send proposal",
  "assigned_to": "uuid",
  "priority": "high",
  "due_date": "2025-11-10"
}
```

**Response:** `201 Created`

### Reports

#### GET /meetings/:id/report
Get personal report for a meeting

**Response:** `200 OK`
```json
{
  "report_id": "uuid",
  "meeting_id": "uuid",
  "participant_id": "uuid",
  "report_content": "# Your Meeting Report\n\n...",
  "metrics": {
    "speaking_time": 420,
    "speaking_percentage": 15.5,
    "contribution_count": 12,
    "questions_asked": 3,
    "engagement_score": 0.82,
    "words_spoken": 850
  },
  "action_items": [
    {
      "title": "Create timeline",
      "priority": "high",
      "due_date": "2025-11-10"
    }
  ],
  "key_contributions": [
    {
      "timestamp": 120,
      "quote": "I suggest we prioritize the mobile app",
      "impact": "high"
    }
  ],
  "created_at": "2025-11-03T10:20:00Z"
}
```

#### GET /meetings/:id/export
Export meeting report

**Query Parameters:**
- `format`: `pdf` | `docx` | `txt` | `json` (default: `pdf`)

**Response:** File download with appropriate Content-Type

#### GET /reports
List all user's meeting reports

**Query Parameters:**
- `from`: ISO date
- `to`: ISO date
- `limit`, `offset`

**Response:** `200 OK`

### Notifications

#### GET /notifications
Get user notifications

**Query Parameters:**
- `is_read`: Filter by read status
- `type`: Filter by notification type
- `limit`, `offset`

**Response:** `200 OK`
```json
{
  "notifications": [
    {
      "id": "uuid",
      "type": "report_ready",
      "title": "Meeting report is ready",
      "message": "Your report for 'Team Standup' is available",
      "data": {
        "meeting_id": "uuid",
        "room_name": "Team Standup"
      },
      "is_read": false,
      "action_url": "/meetings/uuid/report",
      "created_at": "2025-11-03T10:20:00Z"
    }
  ],
  "unread_count": 5,
  "total": 42
}
```

#### PATCH /notifications/:id/read
Mark notification as read

**Response:** `200 OK`

#### POST /notifications/read-all
Mark all notifications as read

**Response:** `200 OK`

### Integration

#### GET /integrations/clickup/authorize
Authorize ClickUp integration

**Response:** `302 Redirect` to ClickUp OAuth

#### POST /integrations/clickup/create-task
Create task in ClickUp from action item

**Request Body:**
```json
{
  "action_item_id": "uuid",
  "list_id": "clickup_list_id"
}
```

**Response:** `201 Created`
```json
{
  "task_id": "clickup_task_id",
  "task_url": "https://app.clickup.com/t/..."
}
```

## Rate Limiting

- **Authentication endpoints**: 5 requests per minute
- **Standard endpoints**: 100 requests per minute
- **Upload endpoints**: 10 requests per minute

Headers included in response:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1699001234
```

## Pagination

List endpoints support pagination:

```
GET /rooms?limit=20&offset=40
```

Response includes pagination metadata:
```json
{
  "data": [...],
  "metadata": {
    "total": 150,
    "limit": 20,
    "offset": 40,
    "has_more": true
  }
}
```

## Webhooks

Subscribe to events via webhooks configuration.

### Events

- `room.created`
- `room.started`
- `room.ended`
- `participant.joined`
- `participant.left`
- `recording.completed`
- `transcript.completed`
- `summary.completed`
- `action_item.created`
- `action_item.completed`

### Webhook Payload

```json
{
  "event": "room.ended",
  "timestamp": "2025-11-03T10:00:00Z",
  "data": {
    "room_id": "uuid",
    "duration": 3600,
    "participants_count": 5
  }
}
```
