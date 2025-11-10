# AI Meeting Analysis Flow

## Overview

Hệ thống AI tự động xử lý ghi âm cuộc họp, chuyển đổi thành văn bản (transcript), phân tích nội dung và tạo báo cáo với action items cho từng người tham gia.

## AI Processing Pipeline

```
Recording → Audio Processing → STT (Whisper) → Transcript
                                                     ↓
                                        Text Analysis (GPT-4)
                                                     ↓
                            ┌────────────────────────┴────────────────────┐
                            ↓                                             ↓
                    Meeting Summary                              Action Items
                    Key Points                                   Assigned Tasks
                    Decisions Made                               Follow-ups
                            ↓                                             ↓
                    ┌───────────────────────────────────────────────────┐
                    │           Generate Personal Reports               │
                    └───────────────────────────────────────────────────┘
                                            ↓
                                    Notify Participants
```

## Complete AI Flow

```mermaid
sequenceDiagram
    participant R as Recording Service
    participant AI as AI Service
    participant W as Whisper API
    participant G as GPT-4 API
    participant DB as Database
    participant CU as ClickUp API
    participant N as Notification Service
    
    R->>AI: Meeting ended, process recording
    Note over R,AI: { recording_id, room_id, file_url }
    
    activate AI
    AI->>AI: Download audio file
    AI->>AI: Preprocess audio<br/>(noise reduction, normalization)
    
    Note over AI: Speech-to-Text Processing
    AI->>W: POST /v1/audio/transcriptions
    Note over W: Model: whisper-1<br/>Language: auto-detect<br/>Response format: verbose_json
    
    W->>W: Transcribe audio
    W-->>AI: Transcript with timestamps
    Note over AI: { text, segments[], language }
    
    AI->>AI: Enhance transcript<br/>(punctuation, speaker diarization)
    AI->>DB: Store transcript
    
    Note over AI: Content Analysis
    AI->>G: Analyze transcript with GPT-4
    Note over G: Prompt: Extract summary,<br/>key points, decisions, action items
    
    G->>G: Process with GPT-4
    G-->>AI: Analysis results
    Note over AI: { summary, key_points,<br/>decisions, action_items[] }
    
    AI->>DB: Store analysis results
    
    Note over AI: Generate Personal Reports
    loop For each participant
        AI->>G: Generate personalized report
        Note over G: Context: transcript + participant_name<br/>Extract: speaking time, contributions,<br/>assigned tasks
        
        G-->>AI: Personal report
        AI->>DB: Store participant report
    end
    
    Note over AI: Optional ClickUp Integration
    alt ClickUp enabled
        loop For each action item
            AI->>CU: Create task
            Note over CU: POST /api/v2/list/:list_id/task
            CU-->>AI: Task created
            AI->>DB: Store task link
        end
    end
    
    AI->>N: Notify participants
    N->>N: Send email notifications
    N->>N: Push in-app notifications
    deactivate AI
```

## Speech-to-Text Process

```mermaid
sequenceDiagram
    participant AI as AI Service
    participant S3 as S3 Storage
    participant W as Whisper API
    participant DB as Database
    
    AI->>S3: Download recording
    S3-->>AI: Audio file (mp3/wav)
    
    AI->>AI: Check audio format
    alt Audio needs conversion
        AI->>AI: Convert to compatible format<br/>(mp3, 16kHz, mono)
    end
    
    AI->>AI: Split audio if > 25MB
    Note over AI: Whisper API limit: 25MB
    
    loop For each audio chunk
        AI->>W: POST /v1/audio/transcriptions
        Note over AI,W: file: audio_chunk<br/>model: whisper-1<br/>language: auto<br/>response_format: verbose_json<br/>temperature: 0<br/>timestamp_granularities: ["word", "segment"]
        
        W->>W: Transcribe with timestamps
        W-->>AI: { text, language, duration,<br/>words[], segments[] }
    end
    
    AI->>AI: Merge chunk transcripts
    AI->>AI: Post-processing
    Note over AI: - Fix common errors<br/>- Add punctuation<br/>- Identify speakers<br/>- Remove filler words (optional)
    
    AI->>DB: INSERT INTO transcripts
    Note over DB: transcript_id, meeting_id,<br/>text, language, words[],<br/>segments[], created_at
    
    AI->>DB: UPDATE meetings<br/>SET has_transcript = true
```

## Speaker Diarization

```mermaid
sequenceDiagram
    participant AI as AI Service
    participant ML as ML Model (pyannote)
    participant G as GPT-4
    participant DB as Database
    
    Note over AI: After getting base transcript
    
    AI->>AI: Extract audio features
    AI->>ML: Diarization request
    Note over ML: pyannote.audio speaker diarization
    
    ML->>ML: Identify speaker segments
    ML-->>AI: Speaker timeline
    Note over AI: [ {start, end, speaker_id}, ... ]
    
    AI->>AI: Match speakers with participants
    Note over AI: Use participant join/leave times
    
    AI->>G: Assign speaker names
    Note over G: Prompt: Given transcript segments<br/>and speakers (Speaker_0, Speaker_1),<br/>identify who said what based on context
    
    G-->>AI: Named transcript
    Note over AI: Each segment tagged with participant
    
    AI->>DB: UPDATE transcript<br/>SET speakers = ?
    
    AI->>DB: Calculate speaking statistics
    Note over DB: - Speaking time per person<br/>- Number of contributions<br/>- Interruptions<br/>- Engagement score
```

## GPT-4 Analysis Process

```mermaid
sequenceDiagram
    participant AI as AI Service
    participant G as GPT-4 API
    participant DB as Database
    
    AI->>AI: Prepare analysis prompt
    Note over AI: Include:<br/>- Full transcript with speakers<br/>- Meeting context (title, participants)<br/>- Expected output format
    
    AI->>G: POST /v1/chat/completions
    Note over G: Model: gpt-4-turbo<br/>Temperature: 0.3<br/>Max tokens: 4000
    
    activate G
    G->>G: Analyze transcript
    Note over G: Extract:<br/>1. Executive summary<br/>2. Key discussion points<br/>3. Decisions made<br/>4. Action items with owners<br/>5. Questions raised<br/>6. Next steps
    
    G-->>AI: Structured analysis
    Note over AI: JSON response with all sections
    deactivate G
    
    AI->>AI: Parse and validate response
    AI->>AI: Enrich action items
    Note over AI: Add priority, estimated time,<br/>deadline suggestions
    
    AI->>DB: INSERT INTO meeting_summaries
    AI->>DB: INSERT INTO action_items
    Note over DB: Each action item with:<br/>- description<br/>- assigned_to<br/>- priority<br/>- status: pending
    
    AI->>DB: INSERT INTO meeting_insights
    Note over DB: Key metrics:<br/>- Sentiment score<br/>- Decision count<br/>- Action item count<br/>- Topics discussed
```

## Personal Report Generation

```mermaid
sequenceDiagram
    participant AI as AI Service
    participant G as GPT-4 API
    participant DB as Database
    
    loop For each participant
        AI->>DB: Get participant data
        Note over DB: - Name, role<br/>- Speaking segments<br/>- Mentions in transcript
        
        AI->>AI: Calculate personal metrics
        Note over AI: - Total speaking time<br/>- % of meeting participation<br/>- Key contributions<br/>- Assigned action items
        
        AI->>AI: Extract relevant segments
        Note over AI: Segments where participant:<br/>- Spoke<br/>- Was mentioned<br/>- Asked questions<br/>- Was assigned tasks
        
        AI->>G: Generate personalized summary
        Note over G: Prompt: Create report for [Name]<br/>including their contributions,<br/>questions, and assigned tasks
        
        G-->>AI: Personalized report
        Note over AI: Markdown formatted with:<br/>- Summary of participation<br/>- Key points raised<br/>- Questions asked<br/>- Tasks assigned<br/>- Follow-up actions
        
        AI->>DB: INSERT INTO participant_reports
        Note over DB: participant_id, meeting_id,<br/>report_content, metrics,<br/>action_items[]
    end
```

## Action Items Extraction

```mermaid
sequenceDiagram
    participant AI as AI Service
    participant G as GPT-4 API
    participant DB as Database
    participant CU as ClickUp API
    
    AI->>G: Extract action items
    Note over G: Prompt: From transcript,<br/>identify all tasks, assignments,<br/>follow-ups, and deadlines
    
    G-->>AI: Raw action items
    Note over AI: [ { task, person, context } ]
    
    AI->>AI: Enhance action items
    loop For each item
        AI->>AI: Parse task details
        Note over AI: - Extract assignee<br/>- Identify deadline<br/>- Determine priority<br/>- Add context
        
        AI->>DB: Match name to user_id
        
        AI->>AI: Classify task type
        Note over AI: Types:<br/>- Action (do something)<br/>- Decision (needs approval)<br/>- Question (needs answer)<br/>- Follow-up (check status)
        
        AI->>DB: INSERT INTO action_items
        Note over DB: {<br/>  meeting_id,<br/>  assigned_to,<br/>  title,<br/>  description,<br/>  type,<br/>  priority,<br/>  due_date,<br/>  status: "pending",<br/>  created_from_transcript: true<br/>}
    end
    
    alt ClickUp integration enabled
        AI->>CU: Create tasks in ClickUp
        Note over CU: Map action items to<br/>ClickUp tasks with:<br/>- Name<br/>- Description<br/>- Assignee<br/>- Due date<br/>- Priority
        
        CU-->>AI: Task URLs
        AI->>DB: UPDATE action_items<br/>SET clickup_task_id, clickup_url
    end
```

## Notification Flow

```mermaid
sequenceDiagram
    participant AI as AI Service
    participant N as Notification Service
    participant E as Email Service
    participant DB as Database
    participant WS as WebSocket
    participant U as Users
    
    AI->>N: Report generation complete
    Note over AI,N: { meeting_id, participants[] }
    
    N->>DB: Get meeting details
    N->>DB: Get participant preferences
    
    loop For each participant
        alt Email notifications enabled
            N->>E: Send email with report
            Note over E: Subject: "Meeting summary: [Title]"<br/>Body: Summary + action items<br/>Attachment: Full report PDF
        end
        
        alt Push notifications enabled
            N->>WS: Send push notification
            WS-->>U: "Your meeting report is ready"
        end
        
        N->>DB: INSERT INTO notifications
        Note over DB: type: "report_ready"<br/>user_id, meeting_id,<br/>read: false
        
        alt Has action items
            N->>E: Send action items email
            Note over E: List of assigned tasks<br/>with deadlines
            
            N->>WS: Send task notifications
            WS-->>U: "You have N new tasks"
        end
    end
```

## API Endpoints

```yaml
# Get Transcript
GET /api/meetings/:id/transcript
  Response:
    transcript_id: string
    text: string
    language: string
    segments: TranscriptSegment[]
    speakers: SpeakerMap

# Get Meeting Summary
GET /api/meetings/:id/summary
  Response:
    summary: string
    key_points: string[]
    decisions: string[]
    topics: string[]
    sentiment: number
    duration: number

# Get Action Items
GET /api/meetings/:id/action-items
  Query:
    assigned_to: user_id (optional)
    status: "pending" | "completed" (optional)
  Response:
    action_items: ActionItem[]

# Get Personal Report
GET /api/meetings/:id/report
  Response:
    report: PersonalReport
    metrics: ParticipationMetrics
    action_items: ActionItem[]

# Update Action Item
PATCH /api/action-items/:id
  Body:
    status: "pending" | "in_progress" | "completed"
    notes: string
  Response: ActionItem

# Regenerate Report
POST /api/meetings/:id/regenerate-report
  Body:
    include_speakers: boolean
    language: string
  Response:
    job_id: string
    status: "queued"

# Export Report
GET /api/meetings/:id/export
  Query:
    format: "pdf" | "docx" | "txt" | "json"
  Response: File download
```

## Database Schema

### transcripts table

```sql
CREATE TABLE transcripts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meeting_id UUID NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    text TEXT NOT NULL,
    language VARCHAR(10),
    segments JSONB, -- Array of {start, end, text, speaker}
    words JSONB, -- Word-level timestamps
    confidence_score FLOAT,
    processing_time INT, -- seconds
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_transcripts_meeting ON transcripts(meeting_id);
CREATE INDEX idx_transcripts_language ON transcripts(language);
```

### meeting_summaries table

```sql
CREATE TABLE meeting_summaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meeting_id UUID NOT NULL UNIQUE REFERENCES meetings(id) ON DELETE CASCADE,
    summary TEXT NOT NULL,
    key_points JSONB, -- Array of strings
    decisions JSONB, -- Array of strings
    topics JSONB, -- Array of strings
    sentiment_score FLOAT, -- -1 to 1
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);
```

### action_items table

```sql
CREATE TABLE action_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meeting_id UUID NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    assigned_to UUID REFERENCES users(id),
    title VARCHAR(500) NOT NULL,
    description TEXT,
    type VARCHAR(50), -- 'action', 'decision', 'question', 'follow_up'
    priority VARCHAR(20) DEFAULT 'medium', -- 'low', 'medium', 'high', 'urgent'
    status VARCHAR(20) DEFAULT 'pending', -- 'pending', 'in_progress', 'completed', 'cancelled'
    due_date DATE,
    clickup_task_id VARCHAR(255),
    clickup_url TEXT,
    transcript_reference TEXT, -- Quote from transcript
    created_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP
);

CREATE INDEX idx_action_items_meeting ON action_items(meeting_id);
CREATE INDEX idx_action_items_assigned ON action_items(assigned_to);
CREATE INDEX idx_action_items_status ON action_items(status);
```

### participant_reports table

```sql
CREATE TABLE participant_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meeting_id UUID NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    participant_id UUID NOT NULL REFERENCES users(id),
    report_content TEXT NOT NULL,
    speaking_time INT, -- seconds
    speaking_percentage FLOAT,
    contribution_count INT,
    questions_asked INT,
    metrics JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT unique_meeting_participant UNIQUE (meeting_id, participant_id)
);

CREATE INDEX idx_reports_meeting ON participant_reports(meeting_id);
CREATE INDEX idx_reports_participant ON participant_reports(participant_id);
```

### meeting_insights table

```sql
CREATE TABLE meeting_insights (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meeting_id UUID NOT NULL UNIQUE REFERENCES meetings(id) ON DELETE CASCADE,
    total_speaking_time INT,
    participant_balance_score FLOAT, -- 0-1, how balanced speaking time was
    question_count INT,
    decision_count INT,
    action_item_count INT,
    dominant_speaker_id UUID REFERENCES users(id),
    topics JSONB,
    sentiment_breakdown JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);
```

## GPT-4 Prompts

### Summary Generation Prompt

```
You are an AI meeting assistant. Analyze the following meeting transcript and provide a comprehensive summary.

Transcript:
{transcript_with_speakers}

Meeting Context:
- Title: {meeting_title}
- Date: {meeting_date}
- Participants: {participant_names}
- Duration: {duration}

Please provide:

1. **Executive Summary** (2-3 paragraphs)
   - Brief overview of the meeting
   - Main purpose and outcomes

2. **Key Discussion Points** (bullet points)
   - Important topics discussed
   - Main arguments or viewpoints

3. **Decisions Made**
   - Clear decisions that were reached
   - Who made the decision and context

4. **Action Items**
   - Specific tasks assigned
   - Person responsible
   - Deadline (if mentioned)
   - Context from discussion

5. **Open Questions**
   - Unresolved issues
   - Questions that need follow-up

6. **Next Steps**
   - What happens after this meeting
   - Future meetings or deadlines

Format the response as JSON with the following structure:
{
  "summary": "...",
  "key_points": ["...", "..."],
  "decisions": [{"decision": "...", "made_by": "...", "context": "..."}],
  "action_items": [{"task": "...", "assigned_to": "...", "deadline": "...", "context": "..."}],
  "open_questions": ["...", "..."],
  "next_steps": ["...", "..."]
}
```

### Personal Report Prompt

```
Create a personalized meeting report for {participant_name}.

Meeting Transcript:
{transcript}

Meeting Summary:
{meeting_summary}

Participant's Contributions:
{participant_segments}

Generate a report including:

1. **Your Participation Summary**
   - Speaking time and percentage
   - Key points you raised

2. **Your Contributions**
   - Important statements you made
   - Questions you asked
   - Ideas you suggested

3. **Action Items Assigned to You**
   - Task description
   - Priority
   - Suggested deadline
   - Context from discussion

4. **Relevant Discussions**
   - Topics where you were mentioned
   - Decisions affecting your work
   - Follow-ups needed

5. **Recommendations**
   - Suggested next actions
   - People to follow up with
   - Additional resources

Write in a professional, concise style. Use bullet points and clear sections.
Format as Markdown.
```

## Error Handling

### Processing Errors

```typescript
interface ProcessingError {
  code: string;
  message: string;
  recovery_action: string;
}

// Common errors
const errors = {
  AUDIO_FORMAT_INVALID: {
    code: "AUDIO_FORMAT_INVALID",
    message: "Audio file format not supported",
    recovery_action: "Convert audio to MP3 or WAV format"
  },
  AUDIO_TOO_LARGE: {
    code: "AUDIO_TOO_LARGE",
    message: "Audio file exceeds 25MB limit",
    recovery_action: "Split audio into smaller chunks"
  },
  TRANSCRIPTION_FAILED: {
    code: "TRANSCRIPTION_FAILED",
    message: "Failed to transcribe audio",
    recovery_action: "Retry with different settings or check audio quality"
  },
  GPT_RATE_LIMIT: {
    code: "GPT_RATE_LIMIT",
    message: "OpenAI API rate limit exceeded",
    recovery_action: "Queue for retry after cooldown period"
  },
  NO_SPEECH_DETECTED: {
    code: "NO_SPEECH_DETECTED",
    message: "No speech found in audio",
    recovery_action: "Check if recording captured audio properly"
  }
}
```

## Performance Optimization

### Processing Time Estimates

| Audio Duration | Whisper STT | GPT-4 Analysis | Total |
|----------------|-------------|----------------|-------|
| 10 minutes | ~30 seconds | ~20 seconds | ~1 min |
| 30 minutes | ~1.5 minutes | ~45 seconds | ~2.5 min |
| 60 minutes | ~3 minutes | ~1.5 minutes | ~5 min |

### Optimization Strategies

1. **Parallel Processing**
   - Process multiple audio chunks simultaneously
   - Generate participant reports in parallel

2. **Caching**
   - Cache GPT-4 responses for similar queries
   - Store intermediate results

3. **Async Queue**
   - Use message queue (RabbitMQ/Redis Queue)
   - Process jobs in background workers

4. **Progressive Results**
   - Stream transcript as it's generated
   - Show preliminary summary before full analysis

## Testing Scenarios

- [ ] Transcribe 10-minute meeting
- [ ] Transcribe 60-minute meeting
- [ ] Handle multiple speakers (2-5)
- [ ] Extract action items correctly
- [ ] Generate accurate summaries
- [ ] Identify speakers correctly
- [ ] Handle poor audio quality
- [ ] Process non-English meetings
- [ ] Handle interruptions and cross-talk
- [ ] Generate personalized reports
- [ ] Create ClickUp tasks
- [ ] Send notifications correctly
