# Recording to AssemblyAI Flow

## 📊 Architecture Overview

**High-level flow:**

1. **Room Ends** → LiveKit sends `room_finished` webhook → Update room status
2. **Recording Ready** → LiveKit sends `recording_finished` webhook → Extract recording URL
3. **Submit to AI** → Call `AIService.SubmitToAssemblyAI(roomID, recordingURL)` 
4. **AssemblyAI Processing** → Transcription + Speaker Diarization (~23s for 30-min audio)
5. **Webhook Callback** → AssemblyAI sends transcript via webhook
6. **Store Transcript** → Create Transcript in DB with speaker labels
7. **Ready for Phase 2** → AI Analysis with Groq LLM

## 🎬 LiveKit Recording Setup

**Prerequisites:**
- LiveKit room created
- Recording enabled in room config
- Recording storage configured (S3/local)
- Webhooks enabled in LiveKit Dashboard

**Required webhook events:**
- `room_finished` - Last participant leaves
- `recording_finished` ⭐ **CRITICAL** - Recording ready for download

**Key data from recording_finished webhook:**
- `room.name` - LiveKit room identifier
- `egress.file.location` - HTTP URL to download recording (required)
- `egress.status` - Should be `EGRESS_COMPLETE`

## 🔧 Architecture Components

**WebhookHandler** (internal/adapter/handler/webhook.go)
- Receives LiveKit webhook events
- Routes based on event type
- `handleRecordingFinished()` - Extracts recording URL, calls AIService

**AIService** (internal/usecase/ai/service.go)
- `SubmitToAssemblyAI(meetingID, recordingURL)` - Submits to AssemblyAI with retries
- `HandleAssemblyAIWebhook()` - Processes AssemblyAI callback, stores transcript

**Repositories:**
- `AIJobRepository` - Tracks transcription jobs (pending, submitted, completed, failed)
- `TranscriptRepository` - Stores final transcripts with speaker info

## 📡 Configuration Required

**Environment variables:**
```
ASSEMBLYAI_API_KEY=aai_xxxxx                          # Get from AssemblyAI dashboard
ASSEMBLYAI_WEBHOOK_SECRET=wh_xxxxx                    # Generate secret for webhook signature
ASSEMBLYAI_WEBHOOK_BASE_URL=https://your-domain/v1/webhooks/assemblyai
```

For local dev with ngrok: Use ngrok URL as ASSEMBLYAI_WEBHOOK_BASE_URL

## 🔄 Processing Pipeline Details

### Phase 1: Recording Extraction
- LiveKit detects recording complete → sends webhook with recording URL
- Backend extracts URL from `egress.file.location`
- Finds room by LiveKit room name
- Calls AIService to submit recording

### Phase 2: AssemblyAI Submission
- Creates AIJob with status: `pending`
- Submits recording URL to AssemblyAI API
- Includes webhook URL for callback
- Gets back external_job_id (transcript ID)
- Updates AIJob status: `submitted`
- Retry logic: Exponential backoff (1s, 2s, 4s, 8s, 15s) - max 3 attempts

### Phase 3: AssemblyAI Processing
- AssemblyAI processes audio asynchronously
- Extracts:
  - **Full transcript text** with word-level timestamps
  - **Speaker diarization** (identifies different speakers automatically)
  - **Language detection**
  - **Confidence scores** per word

### Phase 4: Webhook Callback
- AssemblyAI sends webhook when processing complete
- Includes full transcript with speaker information
- Signature verification (HMAC-SHA256)

### Phase 5: Transcript Storage
- Verify webhook signature against ASSEMBLYAI_WEBHOOK_SECRET
- Parse AssemblyAI response
- Create Transcript record with:
  - Full transcript text
  - Detected language
  - Speaker count
  - Raw JSON response (stored as JSONB)
- Update AIJob: status → `completed`, link to transcript_id

## 🗄️ Database Tables Used

**ai_jobs**
- Tracks transcription job lifecycle
- Fields: id, meeting_id, job_type, status, external_job_id, recording_url, transcript_id
- Statuses: pending → submitted → completed (or failed)

**transcripts**
- Stores final transcripts
- Fields: id, recording_id, meeting_id, text, language, speaker_count, raw_data
- Raw data contains full AssemblyAI response for Phase 2 processing

## ⚡ Key Design Decisions

1. **Webhook-driven** - No polling, fully async
2. **Speaker diarization built-in** - AssemblyAI handles it automatically, no extra API calls
3. **Exponential backoff** - Handles temporary failures gracefully
4. **Signature verification** - Validates AssemblyAI webhooks are genuine
5. **JSONB storage** - Full AssemblyAI response available for future enhancements

## 🚨 Error Handling

**Recording URL missing** → Log warning, return 200 OK, can retry later

**AssemblyAI submission fails** → Exponential backoff retry, then mark as failed with error message

**Webhook signature invalid** → Reject request (400), log security warning

**Processing timeout** (Phase 2 task) → Implement polling or timeout recovery

## 📚 Related Documentation

- [LiveKit Recording Docs](https://docs.livekit.io/realtime/server/recording/)
- [LiveKit Webhooks Docs](https://docs.livekit.io/realtime/server/webhooks/)
- [AssemblyAI Transcription API](https://www.assemblyai.com/docs/transcription)
- [AssemblyAI Speaker Diarization](https://www.assemblyai.com/docs/models/speaker-diarization)

