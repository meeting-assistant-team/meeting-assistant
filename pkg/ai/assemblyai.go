package ai

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/johnquangdev/meeting-assistant/pkg/config"
)

// Client is a minimal AssemblyAI client
type AssemblyAIClient struct {
	apiKey        string
	client        *http.Client
	webhookSecret string
}

// NewAssemblyAIClient creates an AssemblyAI client using the provided config.
// If cfg is nil, falls back to environment variables.
func NewAssemblyAIClient(cfg *config.AssemblyAIConfig) *AssemblyAIClient {
	var apiKey, webhookSecret string
	if cfg != nil {
		apiKey = cfg.APIKey
		webhookSecret = cfg.WebhookSecret
	}
	if apiKey == "" {
		apiKey = os.Getenv("ASSEMBLYAI_API_KEY")
	}
	if webhookSecret == "" {
		webhookSecret = os.Getenv("ASSEMBLYAI_WEBHOOK_SECRET")
	}
	return &AssemblyAIClient{
		apiKey:        apiKey,
		client:        &http.Client{Timeout: 30 * time.Second},
		webhookSecret: webhookSecret,
	}
}

// TranscribeRequest is payload for /v2/transcripts
type TranscribeRequest struct {
	AudioURL          string            `json:"audio_url"`
	SpeakerLabels     bool              `json:"speaker_labels,omitempty"`
	LanguageDetection bool              `json:"language_detection,omitempty"`
	LanguageCode      string            `json:"language_code,omitempty"` // vi for Vietnamese
	AutoChapters      bool              `json:"auto_chapters,omitempty"`
	Summarization     bool              `json:"summarization,omitempty"`
	SummaryModel      string            `json:"summary_model,omitempty"`
	SummaryType       string            `json:"summary_type,omitempty"`
	WebhookURL        string            `json:"webhook_url,omitempty"`
	WebhookAuthHeader string            `json:"webhook_auth_header_name,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
}

// TranscribeResponse is minimal response
type TranscribeResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// Utterance represents a speaker segment from AssemblyAI
type Utterance struct {
	Confidence float64 `json:"confidence"`
	End        int     `json:"end"`
	Start      int     `json:"start"`
	Text       string  `json:"text"`
	Speaker    string  `json:"speaker"`
}

// Chapter represents an auto-generated chapter from AssemblyAI
type AssemblyAIChapter struct {
	Gist     string `json:"gist"`
	Headline string `json:"headline"`
	Summary  string `json:"summary"`
	Start    int    `json:"start"`
	End      int    `json:"end"`
}

// FullTranscriptResponse contains complete transcript data from AssemblyAI
type FullTranscriptResponse struct {
	ID            string              `json:"id"`
	Status        string              `json:"status"`
	Text          string              `json:"text"`
	Summary       string              `json:"summary"`
	Chapters      []AssemblyAIChapter `json:"chapters"`
	Utterances    []Utterance         `json:"utterances"`
	LanguageCode  string              `json:"language_code"`
	Confidence    float64             `json:"confidence"`
	AudioDuration float64             `json:"audio_duration"`
}

// TranscribeAudio requests AssemblyAI to transcribe an external audio URL.
// Returns the transcript job id on success.
func (c *AssemblyAIClient) TranscribeAudio(ctx context.Context, recordingURL, webhookURL, webhookAuthHeader string, metadata map[string]string) (string, error) {
	payload := TranscribeRequest{
		AudioURL:          recordingURL,
		SpeakerLabels:     true,
		LanguageDetection: false, // Use explicit Vietnamese
		LanguageCode:      "vi",  // Vietnamese language
		AutoChapters:      true,
		Summarization:     true,
		SummaryModel:      "informative",
		SummaryType:       "bullets", // bullets or paragraph
		WebhookURL:        webhookURL,
		WebhookAuthHeader: webhookAuthHeader,
		Metadata:          metadata,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.assemblyai.com/v2/transcripts", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	// AssemblyAI expects Authorization header with just the API key (no Bearer prefix)
	req.Header.Set("Authorization", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		// Read response body for detailed error message
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyStr := string(bodyBytes)

		return "", fmt.Errorf("assemblyai returned status %d: %s", resp.StatusCode, bodyStr)
	}

	var tr TranscribeResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", err
	}
	return tr.ID, nil
}

// GetTranscript retrieves a transcript by ID with full details
func (c *AssemblyAIClient) GetTranscript(ctx context.Context, transcriptID string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.assemblyai.com/v2/transcripts/"+transcriptID, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("assemblyai returned status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// VerifyWebhookSignature verifies the webhook signature from AssemblyAI
func (c *AssemblyAIClient) VerifyWebhookSignature(payload []byte, signature string) bool {
	if c.webhookSecret == "" {
		// If no secret configured, skip verification
		return true
	}

	expectedSignature := c.computeSignature(payload)
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// computeSignature computes HMAC-SHA256 signature
func (c *AssemblyAIClient) computeSignature(payload []byte) string {
	h := hmac.New(sha256.New, []byte(c.webhookSecret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}
