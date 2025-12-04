# AI Analysis Flow

## Overview

Automatic processing pipeline that converts meeting recordings to transcripts and generates analysis reports.

## Processing Pipeline

**High-level sequence:**

1. Meeting ends → LiveKit webhook triggers backend
2. Backend downloads recording from LiveKit
3. Send to AssemblyAI for speech-to-text + speaker diarization
4. Receive transcript with speaker labels
5. Send transcript to Groq LLM for analysis
6. Generate summaries, action items, insights
7. Store results in database
8. Notify participants

## Technology Stack

### Speech-to-Text
- **Service**: AssemblyAI API
- **Features**: Automatic speaker diarization, multiple language support
- **Processing time**: ~23 seconds for 30-minute audio
- **Cost**: Free tier includes 185 hours/month

### Text Analysis (LLM)
- **Service**: Groq API (Llama 3.1 70B)
- **Features**: Summarization, action item extraction, sentiment analysis
- **Performance**: 750+ tokens/second (18x faster than GPT-4)
- **Cost**: Free tier: 500 requests/day

## Output Data

Per meeting transcript:
- Full text transcript with timestamps
- Speaker identification and labels
- Confidence scores for each segment

Per meeting analysis:
- Executive summary (2-3 sentences)
- Key discussion points
- Action items with assigned owners
- Sentiment and engagement analysis
- Next meeting recommendations

## Processing Time

- 30-minute meeting: ~23 seconds transcription + ~5 seconds analysis = **28 seconds total**
- 60-minute meeting: ~45 seconds transcription + ~8 seconds analysis = **53 seconds total**

Users see reports in under 2 minutes for typical meetings.

## Error Handling

- Retry logic with exponential backoff
- Job status tracking for failure recovery
- Comprehensive error logging
- Automatic retransmission on failure

## Scaling

No infrastructure scaling needed:
- API-based processing (no servers to manage)
- Monitor AssemblyAI quota (185 hours/month free)
- Monitor Groq quota (500 requests/day free)
- Upgrade to paid tiers when limits reached
