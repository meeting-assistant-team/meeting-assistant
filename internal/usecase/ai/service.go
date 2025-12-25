package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	aai "github.com/AssemblyAI/assemblyai-go-sdk"
	backoff "github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
	pkgai "github.com/johnquangdev/meeting-assistant/pkg/ai"
	"github.com/johnquangdev/meeting-assistant/pkg/config"
	"go.uber.org/zap"

	"github.com/johnquangdev/meeting-assistant/internal/adapter/repository"
	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
)

// Service defines AI orchestration methods
type Service interface {
	StartProcessing(ctx context.Context, meetingID string, recordingURL string) error
	HandleAssemblyAIWebhook(ctx context.Context, payload []byte, signature string) error
	SubmitToAssemblyAI(ctx context.Context, meetingID uuid.UUID, recordingURL string) error
}

type aiService struct {
	aiJobRepo       *repository.AIJobRepository
	transcriptRepo  *repository.TranscriptRepository
	asmClient       *pkgai.AssemblyAIClient
	asmSDKClient    *aai.Client // Official SDK client
	groqClient      *pkgai.GroqClient
	cfg             *config.Config
	logger          *zap.Logger
	uploadSemaphore chan struct{} // Worker pool: limit concurrent uploads
}

// NewAIService constructs a new AI service
func NewAIService(
	aiJobRepo *repository.AIJobRepository,
	transcriptRepo *repository.TranscriptRepository,
	asm *pkgai.AssemblyAIClient,
	groq *pkgai.GroqClient,
	cfg *config.Config,
	logger *zap.Logger,
) Service {
	// Initialize official AssemblyAI SDK client
	asmSDKClient := aai.NewClient(cfg.Assembly.APIKey)

	return &aiService{
		aiJobRepo:       aiJobRepo,
		transcriptRepo:  transcriptRepo,
		asmClient:       asm,
		asmSDKClient:    asmSDKClient,
		groqClient:      groq,
		cfg:             cfg,
		logger:          logger,
		uploadSemaphore: make(chan struct{}, 2), // Max 2 concurrent uploads
	}
}

// StartProcessing starts AI processing for a recording (backward compatible)
func (s *aiService) StartProcessing(ctx context.Context, meetingID string, recordingURL string) error {
	if s.asmClient == nil {
		return fmt.Errorf("assemblyai client not configured")
	}

	// Parse meeting ID
	mid, err := uuid.Parse(meetingID)
	if err != nil {
		return fmt.Errorf("invalid meeting ID: %w", err)
	}

	return s.SubmitToAssemblyAI(ctx, mid, recordingURL)
}

// SubmitToAssemblyAI submits a recording to AssemblyAI for transcription
// Uses official SDK with worker pool to limit concurrent uploads
func (s *aiService) SubmitToAssemblyAI(ctx context.Context, meetingID uuid.UUID, recordingURL string) error {
	if s.asmSDKClient == nil {
		return fmt.Errorf("assemblyai SDK client not configured")
	}

	if recordingURL == "" {
		return fmt.Errorf("recording URL is required")
	}

	// Create new AI job for tracking
	aiJob := entities.NewAIJob(meetingID, entities.AIJobTypeTranscription, recordingURL)
	if err := s.aiJobRepo.CreateAIJob(ctx, aiJob); err != nil {
		return fmt.Errorf("failed to create AI job: %w", err)
	}
	if s.logger != nil {
		s.logger.Info("✅ Created new AI job",
			zap.String("job_id", aiJob.ID.String()),
			zap.String("meeting_id", meetingID.String()),
			zap.String("recording_url", recordingURL),
		)
	}

	// Acquire semaphore slot (worker pool) - blocks if 2 uploads already running
	s.uploadSemaphore <- struct{}{}
	defer func() { <-s.uploadSemaphore }() // Release slot when done

	if s.logger != nil {
		s.logger.Info("🔒 Acquired upload slot",
			zap.String("meeting_id", meetingID.String()),
		)
	}

	// Submit to AssemblyAI with retry logic
	var transcriptID string
	submitFn := func() error {
		if s.logger != nil {
			s.logger.Info("📥 Downloading file from MinIO",
				zap.String("recording_url", recordingURL),
			)
		}

		// Download file from MinIO
		resp, err := http.Get(recordingURL)
		if err != nil {
			return fmt.Errorf("failed to download file: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return fmt.Errorf("MinIO returned status %d", resp.StatusCode)
		}

		if s.logger != nil {
			s.logger.Info("📤 Uploading file to AssemblyAI",
				zap.String("content_type", resp.Header.Get("Content-Type")),
				zap.String("content_length", resp.Header.Get("Content-Length")),
			)
		}

		// Upload to AssemblyAI using official SDK (Upload method on Client)
		uploadURL, err := s.asmSDKClient.Upload(ctx, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to upload to AssemblyAI: %w", err)
		}

		if s.logger != nil {
			s.logger.Info("✅ File uploaded to AssemblyAI",
				zap.String("upload_url", uploadURL),
			)
		}

		// Build webhook URL
		webhookURL := s.cfg.Assembly.WebhookBaseURL
		if webhookURL == "" {
			webhookURL = "https://submaniacally-nonfeeding-adela.ngrok-free.dev/v1/webhooks/assemblyai"
		}

		// Transcribe with Vietnamese language
		params := &aai.TranscriptOptionalParams{
			LanguageCode:  aai.TranscriptLanguageCode("vi"), // Type cast to TranscriptLanguageCode
			SpeakerLabels: aai.Bool(true),
		}

		if s.logger != nil {
			s.logger.Info("🎙️ Starting transcription",
				zap.String("language", "vi"),
				zap.String("webhook_url", webhookURL),
			)
		}

		// Submit transcription request (same API as example)
		transcript, err := s.asmSDKClient.Transcripts.TranscribeFromURL(ctx, uploadURL, params)
		if err != nil {
			if s.logger != nil {
				s.logger.Error("❌ AssemblyAI transcription failed",
					zap.String("meeting_id", meetingID.String()),
					zap.Error(err),
				)
			}
			return err
		}

		// Extract transcript ID (it's a pointer to string)
		if transcript.ID != nil {
			transcriptID = *transcript.ID
		}
		if s.logger != nil {
			s.logger.Info("✅ Transcription job submitted",
				zap.String("meeting_id", meetingID.String()),
				zap.String("transcript_id", transcriptID),
				zap.String("status", string(transcript.Status)),
			)
		}
		return nil
	}

	// Retry logic with exponential backoff
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 2 * time.Second
	bo.MaxElapsedTime = 30 * time.Second
	bo.MaxInterval = 10 * time.Second

	if err := backoff.Retry(submitFn, backoff.WithContext(bo, ctx)); err != nil {
		s.aiJobRepo.MarkJobAsFailed(ctx, aiJob.ID, fmt.Sprintf("failed to submit to AssemblyAI: %v", err))
		if s.logger != nil {
			s.logger.Error("❌ Failed to submit to AssemblyAI after retries",
				zap.String("job_id", aiJob.ID.String()),
				zap.Error(err),
			)
		}
		return err
	}

	// Mark job as submitted with transcript ID
	if err := s.aiJobRepo.MarkJobAsSubmitted(ctx, aiJob.ID, transcriptID); err != nil {
		if s.logger != nil {
			s.logger.Error("❌ Failed to update job status",
				zap.String("job_id", aiJob.ID.String()),
				zap.Error(err),
			)
		}
		return err
	}

	if s.logger != nil {
		s.logger.Info("✅ Successfully submitted to AssemblyAI",
			zap.String("job_id", aiJob.ID.String()),
			zap.String("transcript_id", transcriptID),
		)
	}

	return nil
}

// HandleAssemblyAIWebhook processes AssemblyAI webhook payloads
func (s *aiService) HandleAssemblyAIWebhook(ctx context.Context, payload []byte, signature string) error {
	// Verify signature
	if !s.asmClient.VerifyWebhookSignature(payload, signature) {
		if s.logger != nil {
			s.logger.Warn("invalid webhook signature from AssemblyAI")
		}
		return fmt.Errorf("invalid webhook signature")
	}

	// Parse payload
	var body map[string]interface{}
	if err := json.Unmarshal(payload, &body); err != nil {
		if s.logger != nil {
			s.logger.Error("failed to unmarshal webhook payload", zap.Error(err))
		}
		return fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	transcriptID, ok := body["id"].(string)
	if !ok || transcriptID == "" {
		return fmt.Errorf("transcript ID missing in webhook")
	}

	status, ok := body["status"].(string)
	if !ok {
		status = ""
	}

	if s.logger != nil {
		s.logger.Info("received AssemblyAI webhook",
			zap.String("transcript_id", transcriptID),
			zap.String("status", status),
		)
	}

	// Get AI job by external ID
	aiJob, err := s.aiJobRepo.GetAIJobByExternalID(ctx, transcriptID)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("failed to find AI job", zap.Error(err))
		}
		return fmt.Errorf("failed to find AI job: %w", err)
	}

	if aiJob == nil {
		if s.logger != nil {
			s.logger.Warn("AI job not found for transcript",
				zap.String("transcript_id", transcriptID),
			)
		}
		return fmt.Errorf("AI job not found for transcript %s", transcriptID)
	}

	switch status {
	case "processing":
		// Still processing, update job status
		if err := s.aiJobRepo.UpdateAIJobStatus(ctx, aiJob.ID, entities.AIJobStatusProcessing); err != nil {
			if s.logger != nil {
				s.logger.Error("failed to update job status", zap.Error(err))
			}
		}

	case "completed":
		// Transcription completed, fetch full transcript and store
		if err := s.handleCompletedTranscript(ctx, aiJob, transcriptID); err != nil {
			if s.logger != nil {
				s.logger.Error("❌ Failed to handle completed transcript", zap.Error(err))
			}
			return err
		}

	case "error":
		// Processing failed
		errorMsg := fmt.Sprintf("AssemblyAI error: %v", body["error"])
		if err := s.aiJobRepo.MarkJobAsFailed(ctx, aiJob.ID, errorMsg); err != nil {
			if s.logger != nil {
				s.logger.Error("failed to mark job as failed", zap.Error(err))
			}
		}
		if s.logger != nil {
			s.logger.Error("AssemblyAI reported error", zap.String("error", errorMsg))
		}
	}

	return nil
}

// handleCompletedTranscript fetches full transcript from AssemblyAI API and processes it
func (s *aiService) handleCompletedTranscript(ctx context.Context, aiJob *entities.AIJob, transcriptID string) error {
	if s.logger != nil {
		s.logger.Info("📥 Fetching full transcript from AssemblyAI",
			zap.String("transcript_id", transcriptID),
			zap.String("meeting_id", aiJob.MeetingID.String()),
		)
	}

	// Fetch full transcript using SDK
	transcript, err := s.asmSDKClient.Transcripts.Get(ctx, transcriptID)
	if err != nil {
		return fmt.Errorf("failed to fetch transcript: %w", err)
	}

	if s.logger != nil {
		s.logger.Info("✅ Received full transcript from AssemblyAI",
			zap.String("transcript_id", transcriptID),
			zap.String("status", string(transcript.Status)),
		)
	}

	// Create transcript entity
	transcriptEntity := entities.NewTranscript(aiJob.MeetingID)
	transcriptEntity.ModelUsed = "assemblyai"

	// Extract text
	if transcript.Text != nil {
		transcriptEntity.Text = *transcript.Text
	}

	// Extract language (not a pointer)
	if transcript.LanguageCode != "" {
		transcriptEntity.Language = string(transcript.LanguageCode)
	}

	// Extract confidence
	if transcript.Confidence != nil {
		transcriptEntity.ConfidenceScore = *transcript.Confidence
	}

	// Extract audio duration
	if transcript.AudioDuration != nil {
		transcriptEntity.ProcessingTime = int(*transcript.AudioDuration)
		aiJob.Metadata.DurationSeconds = int(*transcript.AudioDuration)
	}

	// Store transcript in database
	if err := s.transcriptRepo.CreateTranscript(ctx, transcriptEntity); err != nil {
		if s.logger != nil {
			s.logger.Error("❌ Failed to create transcript", zap.Error(err))
		}
		return fmt.Errorf("failed to store transcript: %w", err)
	}

	if s.logger != nil {
		s.logger.Info("✅ Transcript stored in database",
			zap.String("transcript_id", transcriptEntity.ID.String()),
			zap.String("meeting_id", aiJob.MeetingID.String()),
			zap.Int("text_length", len(transcriptEntity.Text)),
		)
	}

	// Store utterances (speaker segments)
	if transcript.Utterances != nil && len(transcript.Utterances) > 0 {
		utterances := make([]entities.TranscriptUtterance, 0, len(transcript.Utterances))
		for _, utt := range transcript.Utterances {
			utterance := entities.TranscriptUtterance{
				TranscriptID: transcriptEntity.ID,
			}
			if utt.Text != nil {
				utterance.Text = *utt.Text
			}
			if utt.Speaker != nil {
				utterance.Speaker = *utt.Speaker
			}
			if utt.Start != nil {
				utterance.StartTime = float64(*utt.Start) / 1000.0 // ms to seconds
			}
			if utt.End != nil {
				utterance.EndTime = float64(*utt.End) / 1000.0
			}
			if utt.Confidence != nil {
				utterance.Confidence = *utt.Confidence
			}
			utterances = append(utterances, utterance)
		}

		if err := s.transcriptRepo.CreateTranscriptUtterances(ctx, utterances); err != nil {
			if s.logger != nil {
				s.logger.Warn("⚠️ Failed to store utterances", zap.Error(err))
			}
		} else {
			if s.logger != nil {
				s.logger.Info("✅ Stored transcript utterances",
					zap.Int("count", len(utterances)),
				)
			}
		}
	}

	// Mark AI job as completed
	if err := s.aiJobRepo.MarkJobAsCompleted(ctx, aiJob.ID, &transcriptEntity.ID); err != nil {
		if s.logger != nil {
			s.logger.Error("⚠️ Failed to mark job as completed", zap.Error(err))
		}
	}

	// Trigger Groq summary generation in background
	go func() {
		bgCtx := context.Background()
		if err := s.generateGroqSummary(bgCtx, transcriptEntity); err != nil {
			if s.logger != nil {
				s.logger.Error("❌ Failed to generate Groq summary",
					zap.String("transcript_id", transcriptEntity.ID.String()),
					zap.Error(err),
				)
			}
		}
	}()

	return nil
}

// generateGroqSummary generates summary using Groq LLM
func (s *aiService) generateGroqSummary(ctx context.Context, transcript *entities.Transcript) error {
	if s.groqClient == nil {
		return fmt.Errorf("groq client not configured")
	}

	if transcript.Text == "" {
		return fmt.Errorf("transcript text is empty")
	}

	if s.logger != nil {
		s.logger.Info("🤖 Generating Groq summary",
			zap.String("transcript_id", transcript.ID.String()),
			zap.Int("text_length", len(transcript.Text)),
		)
	}

	// Generate summary using Groq
	summary, err := s.groqClient.GenerateSummary(ctx, transcript.Text)
	if err != nil {
		return fmt.Errorf("failed to generate summary: %w", err)
	}

	// Update transcript with Groq summary
	transcript.Summary = summary
	if err := s.transcriptRepo.UpdateTranscript(ctx, transcript); err != nil {
		return fmt.Errorf("failed to update transcript with summary: %w", err)
	}

	if s.logger != nil {
		s.logger.Info("✅ Groq summary generated and saved",
			zap.String("transcript_id", transcript.ID.String()),
			zap.Int("summary_length", len(summary)),
		)
	}

	return nil
}

// storeTranscriptFromWebhook stores transcript data from AssemblyAI webhook
func (s *aiService) storeTranscriptFromWebhook(ctx context.Context, aiJob *entities.AIJob, webhookData map[string]interface{}) error {
	// Create transcript entity
	transcript := entities.NewTranscript(aiJob.MeetingID)
	transcript.ModelUsed = "assemblyai"

	// Extract text
	if text, ok := webhookData["text"].(string); ok {
		transcript.Text = text
	}

	// Extract summary
	if summary, ok := webhookData["summary"].(string); ok {
		transcript.Summary = summary
	}

	// Extract chapters
	if chaptersData, ok := webhookData["chapters"].([]interface{}); ok && len(chaptersData) > 0 {
		chapters := make([]entities.Chapter, 0, len(chaptersData))
		for _, chapterData := range chaptersData {
			if chapterMap, ok := chapterData.(map[string]interface{}); ok {
				chapter := entities.Chapter{}
				if gist, ok := chapterMap["gist"].(string); ok {
					chapter.Gist = gist
				}
				if headline, ok := chapterMap["headline"].(string); ok {
					chapter.Headline = headline
				}
				if summary, ok := chapterMap["summary"].(string); ok {
					chapter.Summary = summary
				}
				if start, ok := chapterMap["start"].(float64); ok {
					chapter.Start = start / 1000.0 // Convert ms to seconds
				}
				if end, ok := chapterMap["end"].(float64); ok {
					chapter.End = end / 1000.0 // Convert ms to seconds
				}
				chapters = append(chapters, chapter)
			}
		}
		// Set chapters - GORM will handle JSONB serialization
		transcript.Chapters = chapters
	}

	// Extract language
	if lang, ok := webhookData["language_code"].(string); ok {
		transcript.Language = lang
	}

	// Extract confidence
	if confidence, ok := webhookData["confidence"].(float64); ok {
		transcript.ConfidenceScore = confidence
	}

	// Extract speaker count if available
	if speakers, ok := webhookData["speakers"].(float64); ok {
		aiJob.Metadata.SpeakerCount = int(speakers)
		transcript.SpeakerCount = int(speakers)
		transcript.HasSpeakers = true
	}

	// Store audio duration
	if duration, ok := webhookData["audio_duration"].(float64); ok {
		aiJob.Metadata.DurationSeconds = int(duration)
		transcript.ProcessingTime = int(duration)
	}

	// Create transcript in database
	if err := s.transcriptRepo.CreateTranscript(ctx, transcript); err != nil {
		if s.logger != nil {
			s.logger.Error("failed to create transcript", zap.Error(err))
		}
		return fmt.Errorf("failed to store transcript: %w", err)
	}

	if s.logger != nil {
		s.logger.Info("✅ transcript stored successfully",
			zap.String("transcript_id", transcript.ID.String()),
			zap.String("meeting_id", aiJob.MeetingID.String()),
			zap.Int("text_length", len(transcript.Text)),
			zap.Int("summary_length", len(transcript.Summary)),
		)
	}

	// Extract and store utterances (speaker segments)
	if utterancesData, ok := webhookData["utterances"].([]interface{}); ok && len(utterancesData) > 0 {
		utterances := make([]entities.TranscriptUtterance, 0, len(utterancesData))
		for _, uttData := range utterancesData {
			if uttMap, ok := uttData.(map[string]interface{}); ok {
				utterance := entities.TranscriptUtterance{
					TranscriptID: transcript.ID,
				}
				if text, ok := uttMap["text"].(string); ok {
					utterance.Text = text
				}
				if speaker, ok := uttMap["speaker"].(string); ok {
					utterance.Speaker = speaker
				}
				if start, ok := uttMap["start"].(float64); ok {
					utterance.StartTime = start / 1000.0 // Convert ms to seconds
				}
				if end, ok := uttMap["end"].(float64); ok {
					utterance.EndTime = end / 1000.0 // Convert ms to seconds
				}
				if confidence, ok := uttMap["confidence"].(float64); ok {
					utterance.Confidence = confidence
				}
				utterances = append(utterances, utterance)
			}
		}

		// Store all utterances in one transaction
		if len(utterances) > 0 {
			if err := s.transcriptRepo.CreateTranscriptUtterances(ctx, utterances); err != nil {
				if s.logger != nil {
					s.logger.Error("failed to create transcript utterances", zap.Error(err))
				}
				// Don't fail the whole operation if utterances fail to save
			} else {
				if s.logger != nil {
					s.logger.Info("✅ stored transcript utterances",
						zap.String("transcript_id", transcript.ID.String()),
						zap.Int("utterance_count", len(utterances)),
					)
				}
			}
		}
	}

	// Mark AI job as completed
	if err := s.aiJobRepo.MarkJobAsCompleted(ctx, aiJob.ID, &transcript.ID); err != nil {
		if s.logger != nil {
			s.logger.Error("failed to mark job as completed", zap.Error(err))
		}
		return fmt.Errorf("failed to update job status: %w", err)
	}

	return nil
}
