package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	backoff "github.com/cenkalti/backoff/v4"
	pkgai "github.com/johnquangdev/meeting-assistant/pkg/ai"
	"github.com/johnquangdev/meeting-assistant/pkg/config"

	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
	repo "github.com/johnquangdev/meeting-assistant/internal/domain/repositories"
)

// Service defines AI orchestration methods
type Service interface {
	StartProcessing(ctx context.Context, meetingID string, recordingURL string) error
	HandleAssemblyAIWebhook(ctx context.Context, payload []byte, signature string) error
}

type aiService struct {
	repo       repo.AIRepository
	asmClient  *pkgai.AssemblyAIClient
	groqClient *pkgai.GroqClient
	cfg        *config.Config
}

// NewAIService constructs a new AI service
func NewAIService(r repo.AIRepository, asm *pkgai.AssemblyAIClient, groq *pkgai.GroqClient, cfg *config.Config) Service {
	return &aiService{repo: r, asmClient: asm, groqClient: groq, cfg: cfg}
}

// StartProcessing starts transcription for a recording by creating a job on AssemblyAI
func (s *aiService) StartProcessing(ctx context.Context, meetingID string, recordingURL string) error {
	if s.asmClient == nil {
		return fmt.Errorf("assemblyai client not configured")
	}
	webhookURL := s.cfg.Assembly.WebhookBaseURL
	if webhookURL == "" {
		webhookURL = "/v1/webhooks/assemblyai"
	}

	var jobID string
	operation := func() error {
		id, err := s.asmClient.TranscribeAudio(ctx, recordingURL, webhookURL, "x-assemblyai-signature", map[string]string{"room_id": meetingID})
		if err != nil {
			return err
		}
		jobID = id
		return nil
	}

	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 500 * time.Millisecond
	bo.MaxElapsedTime = 15 * time.Second
	bo.MaxInterval = 5 * time.Second
	err := backoff.Retry(operation, backoff.WithContext(bo, ctx))
	if err != nil {
		_ = s.repo.SaveAIJob(meetingID, "", "failed")
		return err
	}

	// create job record and mark processing
	_ = s.repo.SaveAIJob(meetingID, jobID, "processing")
	return nil
}

// HandleAssemblyAIWebhook processes AssemblyAI webhook payloads
func (s *aiService) HandleAssemblyAIWebhook(ctx context.Context, payload []byte, signature string) error {
	// verify signature
	if !pkgai.VerifyHMAC(s.cfg.Assembly.WebhookSecret, payload, signature) {
		return fmt.Errorf("invalid webhook signature")
	}

	// parse payload (best-effort parsing)
	var body map[string]interface{}
	if err := json.Unmarshal(payload, &body); err != nil {
		return err
	}

	status, _ := body["status"].(string)
	id, _ := body["id"].(string)

	// attempt to get room id from metadata if available
	var roomID string
	if md, ok := body["metadata"].(map[string]interface{}); ok {
		if rid, ok := md["room_id"].(string); ok {
			roomID = rid
		}
	}

	if status == "completed" {
		// try to extract text
		text, _ := body["text"].(string)
		// persist transcript
		t := &entities.Transcript{
			ID:          id,
			RecordingID: id,
			Text:        text,
			ModelUsed:   "assemblyai",
		}
		if err := s.repo.SaveTranscript(t); err != nil {
			return err
		}

		// call Groq to generate summary (best-effort)
		if s.groqClient != nil {
			var summaryText string
			op := func() error {
				st, err := s.groqClient.GenerateSummary(ctx, text)
				if err != nil {
					return err
				}
				summaryText = st
				return nil
			}
			bo := backoff.NewExponentialBackOff()
			bo.InitialInterval = 300 * time.Millisecond
			bo.MaxElapsedTime = 10 * time.Second
			bo.MaxInterval = 3 * time.Second
			if err := backoff.Retry(op, backoff.WithContext(bo, ctx)); err == nil {
				ms := &entities.MeetingSummary{
					ID:               id + "-summary",
					RoomID:           roomID,
					TranscriptID:     t.ID,
					ExecutiveSummary: summaryText,
					ModelUsed:        "groq",
				}
				_ = s.repo.SaveMeetingSummary(ms)
			}
		}

		// save job as completed
		targetRoom := roomID
		if targetRoom == "" {
			targetRoom = t.RoomID
		}
		_ = s.repo.SaveAIJob(targetRoom, id, "completed")
	} else if status == "processing" {
		_ = s.repo.SaveAIJob("", id, "processing")
	}

	return nil
}
