package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	aai "github.com/AssemblyAI/assemblyai-go-sdk"
	backoff "github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
	pkgai "github.com/johnquangdev/meeting-assistant/pkg/ai"
	"github.com/johnquangdev/meeting-assistant/pkg/config"
	"github.com/johnquangdev/meeting-assistant/pkg/jobcontext"
	"go.uber.org/zap"

	"github.com/johnquangdev/meeting-assistant/internal/adapter/repository"
	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
	domainrepo "github.com/johnquangdev/meeting-assistant/internal/domain/repositories"
)

// Service defines AI orchestration methods
type Service interface {
	StartProcessing(ctx context.Context, meetingID string, recordingURL string) error
	HandleAssemblyAIWebhook(ctx context.Context, payload []byte, signature string) error
	SubmitToAssemblyAI(ctx context.Context, jobID uuid.UUID, recordingURL string) error
	StartWorkerPool(ctx context.Context, workerCount int) error
	StopWorkerPool() error
}

type aiService struct {
	aiJobRepo           *repository.AIJobRepository
	transcriptRepo      *repository.TranscriptRepository
	summaryRepo         domainrepo.AIRepository
	recordingRepo       *repository.RecordingRepository
	roomRepo            domainrepo.RoomRepository
	asmClient           *pkgai.AssemblyAIClient
	asmSDKClient        *aai.Client // Official SDK client
	groqClient          *pkgai.GroqClient
	parser              *Parser
	cfg                 *config.Config
	logger              *zap.Logger
	uploadSemaphore     chan struct{} // Worker pool: limit concurrent uploads
	workerStopChan      chan struct{} // Signal workers to stop
	workerWg            sync.WaitGroup
	isWorkerPoolRunning bool
	workerMutex         sync.Mutex
}

// NewAIService constructs a new AI service
func NewAIService(
	aiJobRepo *repository.AIJobRepository,
	transcriptRepo *repository.TranscriptRepository,
	summaryRepo domainrepo.AIRepository,
	recordingRepo *repository.RecordingRepository,
	roomRepo domainrepo.RoomRepository,
	asm *pkgai.AssemblyAIClient,
	groq *pkgai.GroqClient,
	cfg *config.Config,
	logger *zap.Logger,
) Service {
	// Initialize official AssemblyAI SDK client
	asmSDKClient := aai.NewClient(cfg.Assembly.APIKey)

	return &aiService{
		aiJobRepo:           aiJobRepo,
		transcriptRepo:      transcriptRepo,
		summaryRepo:         summaryRepo,
		recordingRepo:       recordingRepo,
		roomRepo:            roomRepo,
		asmClient:           asm,
		asmSDKClient:        asmSDKClient,
		groqClient:          groq,
		parser:              NewParser(),
		cfg:                 cfg,
		logger:              logger,
		uploadSemaphore:     make(chan struct{}, 2), // Max 2 concurrent uploads
		workerStopChan:      make(chan struct{}),
		isWorkerPoolRunning: false,
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

	// Create AI job first
	aiJob := entities.NewAIJob(mid, entities.AIJobTypeTranscription, recordingURL)
	if err := s.aiJobRepo.CreateAIJob(ctx, aiJob); err != nil {
		return fmt.Errorf("failed to create AI job: %w", err)
	}

	return s.SubmitToAssemblyAI(ctx, aiJob.ID, recordingURL)
}

// SubmitToAssemblyAI submits a recording to AssemblyAI for transcription
// Uses official SDK with worker pool to limit concurrent uploads
// Expects job to already exist in database (created by webhook or caller)
func (s *aiService) SubmitToAssemblyAI(ctx context.Context, jobID uuid.UUID, recordingURL string) error {
	if s.asmSDKClient == nil {
		return fmt.Errorf("assemblyai SDK client not configured")
	}

	if recordingURL == "" {
		return fmt.Errorf("recording URL is required")
	}

	// Get existing job from database
	aiJob, err := s.aiJobRepo.GetAIJobByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get AI job: %w", err)
	}
	if aiJob == nil {
		return fmt.Errorf("AI job not found: %s", jobID)
	}

	if s.logger != nil {
		s.logger.Info("🔄 Processing existing AI job",
			zap.String("job_id", aiJob.ID.String()),
			zap.String("meeting_id", aiJob.MeetingID.String()),
			zap.String("recording_url", recordingURL),
			zap.Int("retry_count", aiJob.RetryCount),
		)
	}

	// Acquire semaphore slot (worker pool) - blocks if 2 uploads already running
	s.uploadSemaphore <- struct{}{}
	defer func() { <-s.uploadSemaphore }() // Release slot when done

	if s.logger != nil {
		s.logger.Info("🔒 Acquired upload slot",
			zap.String("meeting_id", aiJob.MeetingID.String()),
		)
	}

	// Submit to AssemblyAI with retry logic
	var transcriptID string
	submitFn := func() error {
		// Trim recording URL to handle old jobs with \n character
		cleanURL := strings.TrimSpace(recordingURL)

		if s.logger != nil {
			s.logger.Info("📥 Downloading file from MinIO",
				zap.String("recording_url", cleanURL),
			)
		}

		// Download file from MinIO
		resp, err := http.Get(cleanURL)
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
			WebhookURL:    &webhookURL, // Tell AssemblyAI where to send webhook when completed
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
					zap.String("meeting_id", aiJob.MeetingID.String()),
					zap.Error(err),
				)
			}
			return err
		}

		// Extract transcript ID (it's a pointer to string)
		if transcript.ID != nil {
			transcriptID = *transcript.ID
		}

		// CRITICAL: Update external_job_id IMMEDIATELY to avoid race with webhook
		// Webhook can arrive within seconds, must have external_job_id in DB first
		if err := s.aiJobRepo.MarkJobAsSubmitted(ctx, aiJob.ID, transcriptID); err != nil {
			if s.logger != nil {
				s.logger.Error("❌ Failed to update external_job_id",
					zap.String("job_id", aiJob.ID.String()),
					zap.Error(err),
				)
			}
			return fmt.Errorf("failed to update external_job_id: %w", err)
		}

		if s.logger != nil {
			s.logger.Info("✅ Transcription job submitted",
				zap.String("meeting_id", aiJob.MeetingID.String()),
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

	// Job already marked as submitted inside submitFn (to avoid webhook race)
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
	// TODO: Enable signature verification when webhook secret is configured properly
	// AssemblyAI doesn't require webhook authentication by default
	// Skip verification for now to allow webhooks through
	/*
		if !s.asmClient.VerifyWebhookSignature(payload, signature) {
			if s.logger != nil {
				s.logger.Warn("invalid webhook signature from AssemblyAI")
			}
			return fmt.Errorf("invalid webhook signature")
		}
	*/

	if s.logger != nil {
		s.logger.Info("📥 Received AssemblyAI webhook (signature verification disabled)")
	}

	// Parse payload
	var body map[string]interface{}
	if err := json.Unmarshal(payload, &body); err != nil {
		if s.logger != nil {
			s.logger.Error("failed to unmarshal webhook payload", zap.Error(err))
		}
		return fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	// Debug: Log raw webhook payload to see structure
	if s.logger != nil {
		s.logger.Info("🔍 AssemblyAI webhook payload",
			zap.String("raw_payload", string(payload)),
			zap.Any("parsed_body", body),
		)
	}

	transcriptID, ok := body["transcript_id"].(string)
	if !ok || transcriptID == "" {
		// Try alternative field name "id"
		transcriptID, ok = body["id"].(string)
		if !ok || transcriptID == "" {
			if s.logger != nil {
				s.logger.Error("❌ Transcript ID missing in webhook",
					zap.Any("available_fields", body),
				)
			}
			return fmt.Errorf("transcript ID missing in webhook")
		}
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

	// Query recording_id from recordings table (get most recent recording for this room)
	recordings, err := s.recordingRepo.FindByRoomID(ctx, aiJob.MeetingID)
	if err == nil && len(recordings) > 0 {
		// Use the most recent recording (already sorted DESC by started_at)
		transcriptEntity.RecordingID = recordings[0].ID.String()
		if s.logger != nil {
			s.logger.Info("✅ Found recording_id for transcript",
				zap.String("recording_id", recordings[0].ID.String()),
			)
		}
	} else if s.logger != nil {
		s.logger.Warn("⚠️ Could not find recording_id",
			zap.String("meeting_id", aiJob.MeetingID.String()),
			zap.Error(err),
		)
	}

	// Query room_id (livekit_room_name) from rooms table
	room, err := s.roomRepo.FindByID(ctx, aiJob.MeetingID)
	if err == nil && room != nil {
		transcriptEntity.RoomID = room.LivekitRoomName
		if s.logger != nil {
			s.logger.Info("✅ Found room_id for transcript",
				zap.String("room_id", room.LivekitRoomName),
			)
		}
	} else if s.logger != nil {
		s.logger.Warn("⚠️ Could not find room_id",
			zap.String("meeting_id", aiJob.MeetingID.String()),
			zap.Error(err),
		)
	}

	// Extract words with timestamps from AssemblyAI response
	if transcript.Words != nil && len(transcript.Words) > 0 {
		words := make([]entities.WordTimestamp, 0, len(transcript.Words))
		for _, w := range transcript.Words {
			word := entities.WordTimestamp{}
			if w.Text != nil {
				word.Word = *w.Text
			}
			if w.Start != nil {
				word.Start = float64(*w.Start) / 1000.0 // ms to seconds
			}
			if w.End != nil {
				word.End = float64(*w.End) / 1000.0
			}
			if w.Confidence != nil {
				word.Confidence = *w.Confidence
			}
			if w.Speaker != nil {
				word.Speaker = *w.Speaker
			}
			words = append(words, word)
		}
		transcriptEntity.Words = words
		if s.logger != nil {
			s.logger.Info("✅ Extracted words from transcript",
				zap.Int("word_count", len(words)),
			)
		}
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

	// Mark AI job as completed and set status to transcript_ready for summary generation
	if err := s.aiJobRepo.UpdateAIJobStatus(ctx, aiJob.ID, entities.AIJobStatusTranscriptReady); err != nil {
		if s.logger != nil {
			s.logger.Error("⚠️ Failed to mark job as transcript_ready", zap.Error(err))
		}
	} else {
		if s.logger != nil {
			s.logger.Info("✅ Job marked as transcript_ready, will be picked up by worker pool",
				zap.String("job_id", aiJob.ID.String()),
				zap.String("transcript_id", transcriptEntity.ID.String()),
			)
		}
	}

	return nil
}

// generateGroqSummary generates summary using Groq LLM (legacy, kept for backward compatibility)
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

// StartWorkerPool starts background workers to process summary jobs
func (s *aiService) StartWorkerPool(ctx context.Context, workerCount int) error {
	s.workerMutex.Lock()
	defer s.workerMutex.Unlock()

	if s.isWorkerPoolRunning {
		return fmt.Errorf("worker pool already running")
	}

	s.isWorkerPoolRunning = true
	s.workerStopChan = make(chan struct{})

	if s.logger != nil {
		s.logger.Info("🚀 Starting AI worker pool",
			zap.Int("worker_count", workerCount),
		)
	}

	// Start worker goroutines
	for i := 0; i < workerCount; i++ {
		s.workerWg.Add(1)
		go s.summaryWorker(ctx, i)
	}

	// Start cleanup routine for zombie jobs
	s.workerWg.Add(1)
	go s.cleanupZombieJobs(ctx)

	// Start worker for pending jobs (submit to AssemblyAI)
	s.workerWg.Add(1)
	go s.pendingJobWorker(ctx)

	// Start worker to cleanup failed jobs
	s.workerWg.Add(1)
	go s.failedJobRetryWorker(ctx)

	// Start worker to poll AssemblyAI for webhook timeouts
	s.workerWg.Add(1)
	go s.webhookTimeoutWorker(ctx)

	return nil
}

// StopWorkerPool gracefully stops all worker goroutines
func (s *aiService) StopWorkerPool() error {
	s.workerMutex.Lock()
	defer s.workerMutex.Unlock()

	if !s.isWorkerPoolRunning {
		return fmt.Errorf("worker pool not running")
	}

	if s.logger != nil {
		s.logger.Info("🛑 Stopping AI worker pool...")
	}

	close(s.workerStopChan)
	s.workerWg.Wait()
	s.isWorkerPoolRunning = false

	if s.logger != nil {
		s.logger.Info("✅ AI worker pool stopped")
	}

	return nil
}

// summaryWorker polls for jobs with transcript_ready status and generates summaries
func (s *aiService) summaryWorker(parentCtx context.Context, workerID int) {
	defer s.workerWg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	if s.logger != nil {
		s.logger.Info("👷 Worker started",
			zap.Int("worker_id", workerID),
		)
	}

	for {
		select {
		case <-s.workerStopChan:
			if s.logger != nil {
				s.logger.Info("👷 Worker stopping",
					zap.Int("worker_id", workerID),
				)
			}
			return

		case <-ticker.C:
			// Poll for jobs
			jobs, err := s.aiJobRepo.GetJobsByStatus(parentCtx, entities.AIJobStatusTranscriptReady)
			if err != nil {
				if s.logger != nil {
					s.logger.Error("❌ Failed to poll jobs",
						zap.Int("worker_id", workerID),
						zap.Error(err),
					)
				}
				continue
			}

			if len(jobs) == 0 {
				continue
			}

			// Process first available job
			job := jobs[0]

			// Atomically claim job by marking as summarizing
			// Only one worker will succeed if multiple workers see the same job
			result := s.aiJobRepo.GetDB().WithContext(parentCtx).
				Model(&entities.AIJob{}).
				Where("id = ? AND status = ?", job.ID, entities.AIJobStatusTranscriptReady).
				Updates(map[string]interface{}{
					"status":     entities.AIJobStatusSummarizing,
					"updated_at": time.Now(),
				})

			if result.Error != nil {
				if s.logger != nil {
					s.logger.Error("❌ Failed to claim job",
						zap.String("job_id", job.ID.String()),
						zap.Error(result.Error),
					)
				}
				continue
			}

			// If no rows affected, another worker already claimed this job
			if result.RowsAffected == 0 {
				if s.logger != nil {
					s.logger.Info("⏭️ Job already claimed by another worker",
						zap.String("job_id", job.ID.String()),
					)
				}
				continue
			}

			if s.logger != nil {
				s.logger.Info("👷 Worker claimed job",
					zap.Int("worker_id", workerID),
					zap.String("job_id", job.ID.String()),
					zap.String("meeting_id", job.MeetingID.String()),
				)
			}

			// Create job context with timeout
			jobCtx, cancel := jobcontext.JobBegin(parentCtx, job.ID, string(job.JobType), workerID)

			// Execute job with retry logic
			err = jobcontext.JobEnd(jobCtx, func(ctx context.Context) error {
				return s.generateMeetingSummary(ctx, &job)
			})

			cancel()

			if err != nil {
				// Job failed after retries
				if s.logger != nil {
					s.logger.Error("❌ Job failed after retries",
						zap.String("job_id", job.ID.String()),
						zap.Error(err),
					)
				}
				s.aiJobRepo.MarkJobAsFailed(parentCtx, job.ID, err.Error())
			} else {
				// Job succeeded
				if s.logger != nil {
					s.logger.Info("✅ Job completed successfully",
						zap.String("job_id", job.ID.String()),
					)
				}
				s.aiJobRepo.UpdateAIJobStatus(parentCtx, job.ID, entities.AIJobStatusCompleted)
			}
		}
	}
}

// generateMeetingSummary generates structured meeting summary using Groq
func (s *aiService) generateMeetingSummary(ctx context.Context, job *entities.AIJob) error {
	startTime := time.Now()

	// Get transcript
	transcript, err := s.transcriptRepo.GetTranscriptByMeetingID(ctx, job.MeetingID)
	if err != nil {
		return fmt.Errorf("failed to get transcript: %w", err)
	}

	if transcript == nil {
		return fmt.Errorf("transcript not found for meeting %s", job.MeetingID)
	}

	// Get utterances (speaker segments) for better analysis
	utterances, err := s.transcriptRepo.GetTranscriptUtterances(ctx, transcript.ID)
	if err != nil {
		return fmt.Errorf("failed to get transcript utterances: %w", err)
	}

	// Format utterances into structured text for Groq
	var formattedTranscript string
	if len(utterances) > 0 {
		// Use speaker-segmented format for better analysis
		var sb strings.Builder
		for _, utt := range utterances {
			// Format: [MM:SS Speaker A]: text
			minutes := int(utt.StartTime) / 60
			seconds := int(utt.StartTime) % 60
			sb.WriteString(fmt.Sprintf("[%02d:%02d %s]: %s\n", minutes, seconds, utt.Speaker, utt.Text))
		}
		formattedTranscript = sb.String()

		if s.logger != nil {
			s.logger.Info("✅ Formatted transcript with speaker segments",
				zap.Int("utterance_count", len(utterances)),
				zap.Int("formatted_length", len(formattedTranscript)),
			)
		}
	} else {
		// Fallback to plain text if no utterances available
		formattedTranscript = transcript.Text

		if s.logger != nil {
			s.logger.Warn("⚠️ No utterances found, using plain text",
				zap.String("transcript_id", transcript.ID.String()),
			)
		}
	}

	// Validate transcript length
	if err := s.parser.ValidateTranscriptLength(formattedTranscript, transcript.ProcessingTime); err != nil {
		// Meeting too short, create minimal summary
		if s.logger != nil {
			s.logger.Info("⚠️ Meeting too short for detailed analysis",
				zap.String("meeting_id", job.MeetingID.String()),
				zap.Error(err),
			)
		}

		return s.createMinimalSummary(ctx, job.MeetingID, transcript.ID, "Meeting was too short to generate detailed analysis.")
	}

	// Detect language mix
	isMixed, primaryLang, ratio := s.parser.DetectLanguageMix(formattedTranscript)
	if s.logger != nil {
		if isMixed {
			s.logger.Info("🌐 Mixed language detected",
				zap.String("primary", primaryLang),
				zap.String("ratio", ratio),
			)
		}
	}

	// Use detected language or fallback to transcript language
	language := primaryLang
	if language == "" || language == "unknown" {
		language = transcript.Language
		if language == "" {
			language = "en" // default fallback
		}
	}

	// Generate structured analysis with Groq
	if s.logger != nil {
		s.logger.Info("🤖 Generating structured analysis with Groq (using speaker segments)",
			zap.String("meeting_id", job.MeetingID.String()),
			zap.String("language", language),
			zap.Int("text_length", len(formattedTranscript)),
			zap.Int("utterance_count", len(utterances)),
		)
	}

	jsonResponse, err := s.groqClient.GenerateStructuredAnalysis(ctx, formattedTranscript, language)
	if err != nil {
		return fmt.Errorf("failed to generate structured analysis: %w", err)
	}

	// Parse JSON response
	analysisResult, err := s.parser.ParseGroqJSONResponse(jsonResponse)
	if err != nil {
		// JSON parsing failed, store raw response and fail
		if s.logger != nil {
			s.logger.Error("❌ Failed to parse Groq JSON response",
				zap.Error(err),
				zap.String("raw_response", jsonResponse[:min(500, len(jsonResponse))]),
			)
		}
		return fmt.Errorf("failed to parse groq response: %w", err)
	}

	// Validate analysis result
	if err := s.parser.ValidateAnalysisResult(analysisResult); err != nil {
		return fmt.Errorf("invalid analysis result: %w", err)
	}

	// Create MeetingSummary entity
	summary := entities.NewMeetingSummary(job.MeetingID, transcript.ID)
	summary.ExecutiveSummary = analysisResult.ExecutiveSummary
	summary.OverallSentiment = analysisResult.OverallSentiment
	summary.EngagementScore = analysisResult.EngagementScore
	summary.ProcessingTime = int(time.Since(startTime).Milliseconds())

	// Marshal JSONB fields
	if keyPoints, err := json.Marshal(analysisResult.KeyPoints); err == nil {
		summary.KeyPoints = keyPoints
	}
	if decisions, err := json.Marshal(analysisResult.Decisions); err == nil {
		summary.Decisions = decisions
	}
	if topics, err := json.Marshal(analysisResult.Topics); err == nil {
		summary.Topics = topics
	}
	if openQuestions, err := json.Marshal(analysisResult.OpenQuestions); err == nil {
		summary.OpenQuestions = openQuestions
	}
	if nextSteps, err := json.Marshal(analysisResult.NextSteps); err == nil {
		summary.NextSteps = nextSteps
	}
	if sentimentBreakdown, err := json.Marshal(analysisResult.SpeakerSentiment); err == nil {
		summary.SentimentBreakdown = sentimentBreakdown
	}

	// Save meeting summary
	if err := s.summaryRepo.SaveMeetingSummary(summary); err != nil {
		return fmt.Errorf("failed to save meeting summary: %w", err)
	}

	if s.logger != nil {
		s.logger.Info("✅ Meeting summary saved",
			zap.String("summary_id", summary.ID.String()),
			zap.String("meeting_id", job.MeetingID.String()),
		)
	}

	// Extract and save action items
	actionItems, err := s.parser.ExtractActionItems(ctx, job.MeetingID, summary.ID, analysisResult)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("⚠️ Failed to extract action items", zap.Error(err))
		}
	} else if len(actionItems) > 0 {
		if err := s.summaryRepo.SaveActionItems(actionItems); err != nil {
			if s.logger != nil {
				s.logger.Warn("⚠️ Failed to save action items", zap.Error(err))
			}
		} else {
			if s.logger != nil {
				s.logger.Info("✅ Action items saved",
					zap.Int("count", len(actionItems)),
				)
			}
		}
	}

	// Update transcript summary field (for backward compatibility)
	transcript.Summary = analysisResult.ExecutiveSummary
	if err := s.transcriptRepo.UpdateTranscript(ctx, transcript); err != nil {
		if s.logger != nil {
			s.logger.Warn("⚠️ Failed to update transcript summary", zap.Error(err))
		}
	}

	return nil
}

// createMinimalSummary creates a minimal summary for very short meetings
func (s *aiService) createMinimalSummary(ctx context.Context, roomID, transcriptID uuid.UUID, message string) error {
	summary := entities.NewMeetingSummary(roomID, transcriptID)
	summary.ExecutiveSummary = message
	summary.KeyPoints = []byte("[]")
	summary.Decisions = []byte("[]")
	summary.Topics = []byte("[]")
	summary.OpenQuestions = []byte("[]")
	summary.NextSteps = []byte("[]")
	summary.SentimentBreakdown = []byte("{}")

	return s.summaryRepo.SaveMeetingSummary(summary)
}

// cleanupZombieJobs resets jobs stuck in summarizing status for >10 minutes
func (s *aiService) cleanupZombieJobs(parentCtx context.Context) {
	defer s.workerWg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.workerStopChan:
			return

		case <-ticker.C:
			jobs, err := s.aiJobRepo.GetJobsByStatus(parentCtx, entities.AIJobStatusSummarizing)
			if err != nil {
				continue
			}

			for _, job := range jobs {
				// Check if job has been running for > 10 minutes
				if job.UpdatedAt.Before(time.Now().Add(-10 * time.Minute)) {
					if s.logger != nil {
						s.logger.Warn("🧹 Cleaning up zombie job",
							zap.String("job_id", job.ID.String()),
							zap.Time("updated_at", job.UpdatedAt),
						)
					}

					// Reset to transcript_ready
					s.aiJobRepo.UpdateAIJobStatus(parentCtx, job.ID, entities.AIJobStatusTranscriptReady)
				}
			}
		}
	}
}

// pendingJobWorker polls for pending/retrying jobs and submits them to AssemblyAI
func (s *aiService) pendingJobWorker(parentCtx context.Context) {
	defer s.workerWg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	if s.logger != nil {
		s.logger.Info("👷 Pending job worker started")
	}

	for {
		select {
		case <-s.workerStopChan:
			if s.logger != nil {
				s.logger.Info("👷 Pending job worker stopping")
			}
			return

		case <-ticker.C:
			// Poll for pending/retrying jobs
			jobs, err := s.aiJobRepo.GetJobsForProcessing(parentCtx, 5)
			if err != nil {
				if s.logger != nil {
					s.logger.Error("❌ Failed to poll pending jobs", zap.Error(err))
				}
				continue
			}

			if len(jobs) == 0 {
				continue
			}

			if s.logger != nil {
				s.logger.Info("📋 Found pending/retrying jobs",
					zap.Int("count", len(jobs)),
				)
			}

			// Process each job
			for _, job := range jobs {
				// Atomically claim the job by marking as submitted (prevent other workers from picking it up)
				// Use WHERE clause with current status to ensure only one worker claims it
				result := s.aiJobRepo.GetDB().WithContext(parentCtx).
					Model(&entities.AIJob{}).
					Where("id = ? AND status = ?", job.ID, entities.AIJobStatusPending).
					Updates(map[string]interface{}{
						"status":     entities.AIJobStatusSubmitted,
						"started_at": time.Now(),
						"updated_at": time.Now(),
					})

				if result.Error != nil {
					if s.logger != nil {
						s.logger.Error("❌ Failed to claim job",
							zap.String("job_id", job.ID.String()),
							zap.Error(result.Error),
						)
					}
					continue
				}

				// If no rows affected, another worker already claimed this job
				if result.RowsAffected == 0 {
					if s.logger != nil {
						s.logger.Info("⏭️ Job already claimed by another worker",
							zap.String("job_id", job.ID.String()),
						)
					}
					continue
				}

				// Successfully claimed the job, now submit to AssemblyAI
				if s.logger != nil {
					s.logger.Info("📤 Worker claimed job, submitting to AssemblyAI",
						zap.String("job_id", job.ID.String()),
						zap.String("status", string(job.Status)),
						zap.Int("retry_count", job.RetryCount),
					)
				}

				// Submit to AssemblyAI using existing job
				if err := s.SubmitToAssemblyAI(parentCtx, job.ID, job.RecordingURL); err != nil {
					if s.logger != nil {
						s.logger.Error("❌ Failed to submit job",
							zap.String("job_id", job.ID.String()),
							zap.Error(err),
						)
					}
					// SubmitToAssemblyAI already calls MarkJobAsFailed which handles retry logic
				}
			}
		}
	}
}

// failedJobRetryWorker periodically checks for permanently failed jobs and logs them
func (s *aiService) failedJobRetryWorker(parentCtx context.Context) {
	defer s.workerWg.Done()

	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	if s.logger != nil {
		s.logger.Info("👷 Failed job retry worker started")
	}

	for {
		select {
		case <-s.workerStopChan:
			if s.logger != nil {
				s.logger.Info("👷 Failed job retry worker stopping")
			}
			return

		case <-ticker.C:
			// Get permanently failed jobs (retry_count >= max_retries)
			var failedJobs []entities.AIJob
			if err := s.aiJobRepo.GetDB().WithContext(parentCtx).
				Where("status = ? AND retry_count >= max_retries", entities.AIJobStatusFailed).
				Find(&failedJobs).Error; err != nil {
				continue
			}

			if len(failedJobs) > 0 {
				if s.logger != nil {
					s.logger.Warn("⚠️ Permanently failed jobs found (exceeded max retries)",
						zap.Int("count", len(failedJobs)),
					)
					for _, job := range failedJobs {
						errorMsg := ""
						if job.LastError != nil {
							errorMsg = *job.LastError
						}
						s.logger.Warn("💀 Dead job",
							zap.String("job_id", job.ID.String()),
							zap.String("meeting_id", job.MeetingID.String()),
							zap.Int("retry_count", job.RetryCount),
							zap.String("last_error", errorMsg),
						)
					}
				}
			}
		}
	}
}

// webhookTimeoutWorker polls AssemblyAI API for jobs stuck in submitted status (webhook timeout)
func (s *aiService) webhookTimeoutWorker(parentCtx context.Context) {
	defer s.workerWg.Done()

	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	if s.logger != nil {
		s.logger.Info("👷 Webhook timeout worker started")
	}

	for {
		select {
		case <-s.workerStopChan:
			if s.logger != nil {
				s.logger.Info("👷 Webhook timeout worker stopping")
			}
			return

		case <-ticker.C:
			// Find jobs stuck in submitted status for > 10 minutes
			var stuckJobs []entities.AIJob
			cutoffTime := time.Now().Add(-10 * time.Minute)

			if err := s.aiJobRepo.GetDB().WithContext(parentCtx).
				Where("status = ? AND updated_at < ?", entities.AIJobStatusSubmitted, cutoffTime).
				Find(&stuckJobs).Error; err != nil {
				if s.logger != nil {
					s.logger.Error("❌ Failed to query stuck jobs", zap.Error(err))
				}
				continue
			}

			if len(stuckJobs) == 0 {
				continue
			}

			if s.logger != nil {
				s.logger.Warn("⏰ Found jobs stuck in submitted status (webhook timeout)",
					zap.Int("count", len(stuckJobs)),
				)
			}

			// Poll AssemblyAI API for each stuck job
			for _, job := range stuckJobs {
				if job.ExternalJobID == nil || *job.ExternalJobID == "" {
					if s.logger != nil {
						s.logger.Warn("⚠️ Job has no external ID, marking as failed",
							zap.String("job_id", job.ID.String()),
						)
					}
					s.aiJobRepo.MarkJobAsFailed(parentCtx, job.ID, "no external transcript ID")
					continue
				}

				transcriptID := *job.ExternalJobID

				if s.logger != nil {
					s.logger.Info("🔍 Polling AssemblyAI for stuck job",
						zap.String("job_id", job.ID.String()),
						zap.String("transcript_id", transcriptID),
						zap.Duration("stuck_for", time.Since(job.UpdatedAt)),
					)
				}

				// Get transcript status from AssemblyAI
				transcript, err := s.asmSDKClient.Transcripts.Get(parentCtx, transcriptID)
				if err != nil {
					if s.logger != nil {
						s.logger.Error("❌ Failed to poll AssemblyAI",
							zap.String("transcript_id", transcriptID),
							zap.Error(err),
						)
					}
					// Don't mark as failed yet, might be temporary API error
					continue
				}

				// Check transcript status
				switch transcript.Status {
				case aai.TranscriptStatusCompleted:
					// Webhook was missed, process it now
					if s.logger != nil {
						s.logger.Info("✅ Transcript completed (webhook missed), processing now",
							zap.String("job_id", job.ID.String()),
							zap.String("transcript_id", transcriptID),
						)
					}

					// Process the completed transcript (same as webhook handler)
					if err := s.handleCompletedTranscript(parentCtx, &job, transcriptID); err != nil {
						if s.logger != nil {
							s.logger.Error("❌ Failed to process completed transcript",
								zap.String("job_id", job.ID.String()),
								zap.Error(err),
							)
						}
						s.aiJobRepo.MarkJobAsFailed(parentCtx, job.ID, fmt.Sprintf("failed to process transcript: %v", err))
					}

				case aai.TranscriptStatusError:
					// Transcription failed
					errorMsg := "AssemblyAI transcription failed"
					if transcript.Error != nil {
						errorMsg = fmt.Sprintf("AssemblyAI error: %s", *transcript.Error)
					}
					if s.logger != nil {
						s.logger.Error("❌ AssemblyAI reported error",
							zap.String("job_id", job.ID.String()),
							zap.String("transcript_id", transcriptID),
							zap.String("error", errorMsg),
						)
					}
					s.aiJobRepo.MarkJobAsFailed(parentCtx, job.ID, errorMsg)

				case aai.TranscriptStatusQueued, aai.TranscriptStatusProcessing:
					// Still processing, update timestamp to reset timeout
					if s.logger != nil {
						s.logger.Info("⏳ Transcript still processing",
							zap.String("job_id", job.ID.String()),
							zap.String("transcript_id", transcriptID),
							zap.String("status", string(transcript.Status)),
						)
					}
					// Update timestamp to give it more time
					s.aiJobRepo.GetDB().WithContext(parentCtx).
						Model(&entities.AIJob{}).
						Where("id = ?", job.ID).
						Update("updated_at", time.Now())

				default:
					if s.logger != nil {
						s.logger.Warn("⚠️ Unknown transcript status",
							zap.String("job_id", job.ID.String()),
							zap.String("transcript_id", transcriptID),
							zap.String("status", string(transcript.Status)),
						)
					}
				}
			}
		}
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
