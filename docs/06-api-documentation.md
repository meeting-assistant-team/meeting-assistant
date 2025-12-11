# API Documentation

**Note**: Complete API reference with examples available in `postman_testing.md`

## Base URLs

- Development: `http://localhost:8080/api/v1`
- Production: `https://api.meetingassistant.com/api/v1`

## Authentication

All endpoints (except auth) require JWT Bearer token:
```
Authorization: Bearer {access_token}
```

## Core Endpoint Categories

### Authentication (`/auth`)
- POST `/auth/google` - Google OAuth login
- POST `/auth/callback` - OAuth callback handler
- POST `/auth/refresh` - Refresh access token
- POST `/auth/logout` - Logout and revoke tokens

### Rooms (`/rooms`)
- POST `/rooms` - Create meeting room
- GET `/rooms` - List user's rooms
- GET `/rooms/:id` - Get room details
- POST `/rooms/:id/join` - Request to join room
- POST `/rooms/:id/admit` - Host admits participant
- DELETE `/rooms/:id` - End/delete room

### Participants (`/rooms/:roomId/participants`)
- GET `/participants` - List room participants
- POST `/participants/:id/admit` - Approve participant
- POST `/participants/:id/reject` - Reject participant
- POST `/participants/:id/remove` - Remove participant

### AI Processing (`/ai`)
- POST `/ai/transcribe` - Submit recording for transcription
- GET `/ai/transcript/:id` - Get transcript status/results
- POST `/ai/analyze` - Analyze transcript
- GET `/ai/summary/:id` - Get analysis summary

### Health Check
- GET `/health` - Service health status

## Response Format

All responses include:
```json
{
  "success": true/false,
  "data": {},
  "error": {
    "code": "ERROR_CODE",
    "message": "Error description"
  },
  "metadata": {
    "timestamp": "2025-01-01T00:00:00Z",
    "request_id": "uuid"
  }
}
```

## Error Codes

Common error codes:
- `BAD_REQUEST` (400): Invalid request parameters
- `UNAUTHORIZED` (401): Missing or invalid authentication
- `FORBIDDEN` (403): Insufficient permissions
- `NOT_FOUND` (404): Resource not found
- `CONFLICT` (409): Resource already exists
- `INTERNAL_SERVER_ERROR` (500): Server error

## For Detailed Examples

See `postman_testing.md` with complete request/response examples for all endpoints.
