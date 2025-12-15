package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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
	aiJobRepo      *repository.AIJobRepository
	transcriptRepo *repository.TranscriptRepository
	asmClient      *pkgai.AssemblyAIClient
	groqClient     *pkgai.GroqClient
	cfg            *config.Config
	logger         *zap.Logger
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
	return &aiService{
		aiJobRepo:      aiJobRepo,
		transcriptRepo: transcriptRepo,
		asmClient:      asm,
		groqClient:     groq,
		cfg:            cfg,
		logger:         logger,
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
func (s *aiService) SubmitToAssemblyAI(ctx context.Context, meetingID uuid.UUID, recordingURL string) error {
	if s.asmClient == nil {
		return fmt.Errorf("assemblyai client not configured")
	}

	if recordingURL == "" {
		return fmt.Errorf("recording URL is required")
	}

	// Create new AI job for each recording (each egress can have multiple files - audio, video, etc.)
	// We should create separate jobs to track each transcription properly
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

	// Submit to AssemblyAI with retry logic
	var externalJobID string
	submitFn := func() error {
		// Build webhook URL
		webhookURL := s.cfg.Assembly.WebhookBaseURL
		if webhookURL == "" {
			webhookURL = "https://api.example.com/v1/webhooks/assemblyai"
		}

		if s.logger != nil {
			s.logger.Info("📤 Submitting to AssemblyAI",
				zap.String("meeting_id", meetingID.String()),
				zap.String("recording_url", recordingURL),
				zap.String("webhook_url", webhookURL),
			)
		}

		id, err := s.asmClient.TranscribeAudio(ctx, recordingURL, webhookURL, "x-assemblyai-signature", map[string]string{
			"meeting_id": meetingID.String(),
		})
		if err != nil {
			if s.logger != nil {
				s.logger.Error("❌ AssemblyAI submission failed",
					zap.String("meeting_id", meetingID.String()),
					zap.Error(err),
				)
			}
			return err
		}
		externalJobID = id
		if s.logger != nil {
			s.logger.Info("✅ AssemblyAI accepted job",
				zap.String("meeting_id", meetingID.String()),
				zap.String("external_job_id", externalJobID),
			)
		}
		return nil
	}

	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 1 * time.Second
	bo.MaxElapsedTime = 15 * time.Second
	bo.MaxInterval = 5 * time.Second

	if err := backoff.Retry(submitFn, backoff.WithContext(bo, ctx)); err != nil {
		s.aiJobRepo.MarkJobAsFailed(ctx, aiJob.ID, fmt.Sprintf("failed to submit to AssemblyAI: %v", err))
		if s.logger != nil {
			s.logger.Error("failed to submit audio to AssemblyAI",
				zap.String("job_id", aiJob.ID.String()),
				zap.Error(err),
			)
		}
		return err
	}

	// Mark job as submitted
	if err := s.aiJobRepo.MarkJobAsSubmitted(ctx, aiJob.ID, externalJobID); err != nil {
		if s.logger != nil {
			s.logger.Error("failed to update job status",
				zap.String("job_id", aiJob.ID.String()),
				zap.Error(err),
			)
		}
		return err
	}

	if s.logger != nil {
		s.logger.Info("submitted to AssemblyAI successfully",
			zap.String("job_id", aiJob.ID.String()),
			zap.String("external_job_id", externalJobID),
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
		// Transcription completed, parse and store transcript
		if err := s.storeTranscriptFromWebhook(ctx, aiJob, body); err != nil {
			if s.logger != nil {
				s.logger.Error("failed to store transcript", zap.Error(err))
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
