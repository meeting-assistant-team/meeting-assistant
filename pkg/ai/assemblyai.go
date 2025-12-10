package ai

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	WebhookURL        string            `json:"webhook_url,omitempty"`
	WebhookAuthHeader string            `json:"webhook_auth_header_name,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
}

// TranscribeResponse is minimal response
type TranscribeResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// TranscribeAudio requests AssemblyAI to transcribe an external audio URL.
// Returns the transcript job id on success.
func (c *AssemblyAIClient) TranscribeAudio(ctx context.Context, recordingURL, webhookURL, webhookAuthHeader string, metadata map[string]string) (string, error) {
	payload := TranscribeRequest{
		AudioURL:          recordingURL,
		SpeakerLabels:     true,
		LanguageDetection: true,
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
	req.Header.Set("Authorization", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("assemblyai returned status %d", resp.StatusCode)
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
