# Room Management Flow

## Overview

Manage meeting rooms with host approval workflow for participant admission.

## Room States

- **Scheduled**: Room created but not started
- **Active**: Meeting in progress
- **Ended**: Meeting completed
- **Archived**: Old meetings (for reporting)

## Create Room

Host can create a meeting room with:
- Name and description
- Maximum participant limit
- Recording settings
- Meeting type (training, standup, etc)

Automatically generates:
- Shareable join link
- LiveKit room on the WebRTC server
- Database entry with metadata

## Join Room

**Participant Flow:**
1. Click shareable join link
2. Authenticate if needed
3. Enter waiting room
4. Host receives notification
5. Host approves/denies admission
6. On approval, participant joins meeting

**Host automatically joins** (no approval needed for host)

## Access Control

- **Host**: Full control (create, start, end, remove participants, record)
- **Participant**: Join (if approved), share audio/video, chat
- **Waiting participants**: Cannot see/hear meeting content

## Room Lifecycle

1. Create room → Scheduled state
2. Host starts meeting → Active state
3. Recording begins (if enabled)
4. Participants join (with host approval)
5. Host ends meeting → Ended state
6. Recording stops and starts processing

## LiveKit Integration

Backend creates corresponding LiveKit rooms for:
- Real-time audio/video routing
- Media streaming
- Recording capture
- Webhook notifications

## End Meeting

Host can end meeting which:
- Stops recording
- Disconnects all participants
- Closes LiveKit room
- Marks room as ended in database

## Security

- All participants require authentication
- Host approval required for all joins
- Room access tokens are short-lived
- Webhook signature validation for LiveKit events
