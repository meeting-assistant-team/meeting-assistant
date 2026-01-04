package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"golang.org/x/time/rate"

	"github.com/johnquangdev/meeting-assistant/pkg/config"
)

// GroqClient is a minimal client for Groq API calls used for LLM analysis
type GroqClient struct {
	apiKey      string
	baseURL     string
	client      *http.Client
	rateLimiter *rate.Limiter
}

// NewGroqClient creates a Groq client using values from the provided config.
// Pass a nil config to fall back to environment variables.
func NewGroqClient(cfg *config.GroqConfig) *GroqClient {
	var apiKey string
	if cfg != nil {
		apiKey = cfg.APIKey
	}
	if apiKey == "" {
		apiKey = os.Getenv("GROQ_API_KEY")
	}

	var base string
	if cfg != nil && cfg.BaseURL != "" {
		base = cfg.BaseURL
	} else {
		base = os.Getenv("GROQ_API_URL")
		if base == "" {
			base = "https://api.groq.com"
		}
	}

	// Rate limiter: 30 requests per minute = 1 request per 2 seconds
	limiter := rate.NewLimiter(rate.Every(2*time.Second), 1)

	return &GroqClient{
		apiKey:      apiKey,
		baseURL:     base,
		client:      &http.Client{Timeout: 60 * time.Second}, // Increased for structured analysis
		rateLimiter: limiter,
	}
}

// ChatRequest is the shape for chat completion requests
type ChatRequest struct {
	Model       string      `json:"model,omitempty"`
	Messages    interface{} `json:"messages,omitempty"`
	Temperature float64     `json:"temperature,omitempty"`
	MaxTokens   int         `json:"max_tokens,omitempty"`
}

// ChatResponse is a minimal response shape
type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// GenerateSummary sends the transcript to Groq and returns the assistant content
func (g *GroqClient) GenerateSummary(ctx context.Context, transcript string) (string, error) {
	// Wait for rate limit slot
	if err := g.rateLimiter.Wait(ctx); err != nil {
		return "", fmt.Errorf("rate limit wait cancelled: %w", err)
	}

	prompt := fmt.Sprintf("Analyze the following transcript and return a JSON summary:\n\n%s", transcript)

	reqBody := ChatRequest{
		Model:       "llama-3.3-70b-versatile",
		Messages:    []map[string]string{{"role": "user", "content": prompt}},
		Temperature: 0.3,
		MaxTokens:   8000,
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	endpoint := g.baseURL + "/openai/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+g.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		retryAfter := resp.Header.Get("Retry-After")
		return "", fmt.Errorf("rate limit exceeded (429), retry after: %s", retryAfter)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("groq returned status %d", resp.StatusCode)
	}

	var cr ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return "", err
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("empty response from groq")
	}
	return cr.Choices[0].Message.Content, nil
}

// GenerateStructuredAnalysis generates comprehensive meeting analysis with structured JSON output
func (g *GroqClient) GenerateStructuredAnalysis(ctx context.Context, transcript string, language string) (string, error) {
	// Wait for rate limit slot
	if err := g.rateLimiter.Wait(ctx); err != nil {
		return "", fmt.Errorf("rate limit wait cancelled: %w", err)
	}

	// Clean transcript before analysis
	cleanedTranscript := CleanTranscript(transcript)

	// Truncate if too long (max ~24000 characters for 6000 tokens input limit)
	if len(cleanedTranscript) > 24000 {
		cleanedTranscript = cleanedTranscript[:24000] + "\n\n[... Transcript truncated due to length limit ...]"
	}

	// Build language-appropriate prompt
	var systemPrompt, userPrompt string

	if language == "vi" || strings.Contains(strings.ToLower(language), "vietnam") {
		systemPrompt = `Bạn là một AI chuyên phân tích cuộc họp. Nhiệm vụ của bạn là phân tích transcript và trả về một JSON summary có cấu trúc.

Yêu cầu output JSON schema:
{
  "executive_summary": "Tóm tắt tổng quan ngắn gọn về cuộc họp (2-3 câu)",
  "key_points": [
    {
      "text": "Nội dung key point",
      "timestamp_seconds": 120,
      "mentioned_by_speaker": "Speaker A",
      "importance": "high"
    }
  ],
  "decisions": [
    {
      "decision_text": "Quyết định được đưa ra",
      "owner": "Speaker B",
      "timestamp_seconds": 300,
      "impact": "high"
    }
  ],
  "topics": ["topic1", "topic2"],
  "open_questions": ["Câu hỏi chưa được giải đáp"],
  "next_steps": [
    {
      "description": "Mô tả next step",
      "owner": "Speaker A",
      "due_date_mentioned": "tuần sau",
      "priority": "high"
    }
  ],
  "action_items": [
    {
      "title": "Tiêu đề task",
      "description": "Chi tiết task",
      "assigned_to": "Speaker C",
      "type": "action",
      "priority": "medium",
      "transcript_reference": "Quote từ transcript",
      "timestamp_in_meeting": 450
    }
  ],
  "overall_sentiment": 0.7,
  "speaker_sentiment": {
    "Speaker A": 0.8,
    "Speaker B": 0.6
  },
  "engagement_score": 0.75,
  "participant_balance": {
    "Speaker A": {
      "speaking_time_seconds": 300,
      "speaking_percentage": 50.0,
      "turn_count": 15,
      "sentiment": 0.8,
      "engagement_level": "high"
    }
  }
}

Lưu ý:
- Sentiment score từ -1.0 (rất tiêu cực) đến 1.0 (rất tích cực)
- Importance/Priority: low, medium, high, urgent
- Engagement level: low, medium, high
- Bỏ qua filler words (ừm, à, um, uh)
- Trả về ONLY valid JSON, không có text giải thích thêm`

		userPrompt = fmt.Sprintf("Phân tích transcript cuộc họp sau:\n\n%s", cleanedTranscript)
	} else {
		systemPrompt = `You are an AI specialized in meeting analysis. Your task is to analyze the transcript and return a structured JSON summary.

Required JSON schema:
{
  "executive_summary": "Brief overview of the meeting (2-3 sentences)",
  "key_points": [
    {
      "text": "Key point content",
      "timestamp_seconds": 120,
      "mentioned_by_speaker": "Speaker A",
      "importance": "high"
    }
  ],
  "decisions": [
    {
      "decision_text": "Decision made",
      "owner": "Speaker B",
      "timestamp_seconds": 300,
      "impact": "high"
    }
  ],
  "topics": ["topic1", "topic2"],
  "open_questions": ["Unresolved questions"],
  "next_steps": [
    {
      "description": "Next step description",
      "owner": "Speaker A",
      "due_date_mentioned": "next week",
      "priority": "high"
    }
  ],
  "action_items": [
    {
      "title": "Task title",
      "description": "Task details",
      "assigned_to": "Speaker C",
      "type": "action",
      "priority": "medium",
      "transcript_reference": "Quote from transcript",
      "timestamp_in_meeting": 450
    }
  ],
  "overall_sentiment": 0.7,
  "speaker_sentiment": {
    "Speaker A": 0.8,
    "Speaker B": 0.6
  },
  "engagement_score": 0.75,
  "participant_balance": {
    "Speaker A": {
      "speaking_time_seconds": 300,
      "speaking_percentage": 50.0,
      "turn_count": 15,
      "sentiment": 0.8,
      "engagement_level": "high"
    }
  }
}

Notes:
- Sentiment score from -1.0 (very negative) to 1.0 (very positive)
- Importance/Priority: low, medium, high, urgent
- Engagement level: low, medium, high
- Ignore filler words (um, uh, like, you know)
- Return ONLY valid JSON, no additional explanatory text`

		userPrompt = fmt.Sprintf("Analyze the following meeting transcript:\n\n%s", cleanedTranscript)
	}

	reqBody := ChatRequest{
		Model: "llama-3.3-70b-versatile",
		Messages: []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		Temperature: 0.3,
		MaxTokens:   8000,
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := g.baseURL + "/openai/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(b))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+g.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call groq API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		retryAfter := resp.Header.Get("Retry-After")
		return "", fmt.Errorf("rate limit exceeded (429), retry after: %s", retryAfter)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("groq returned status %d", resp.StatusCode)
	}

	var cr ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("empty response from groq")
	}

	return cr.Choices[0].Message.Content, nil
}

// CleanTranscript removes filler words, repeated phrases, and excess whitespace
func CleanTranscript(transcript string) string {
	text := transcript

	// Remove Vietnamese filler words
	vnFillers := []string{"ừm", "àm", "ờ", "à", "ủa", "ơ", "ê"}
	for _, filler := range vnFillers {
		re := regexp.MustCompile(`\b` + regexp.QuoteMeta(filler) + `\b`)
		text = re.ReplaceAllString(text, "")
	}

	// Remove English filler words
	enFillers := []string{"um", "uh", "ah", "like", "you know", "I mean", "sort of", "kind of"}
	for _, filler := range enFillers {
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(filler) + `\b`)
		text = re.ReplaceAllString(text, "")
	}

	// Remove repeated words (more than 3 consecutive occurrences)
	text = removeExcessiveRepeats(text, 3)

	// Collapse multiple spaces into single space
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	// Remove empty lines
	lines := strings.Split(text, "\n")
	cleanedLines := make([]string, 0)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleanedLines = append(cleanedLines, line)
		}
	}

	return strings.Join(cleanedLines, "\n")
}

// removeExcessiveRepeats removes words that repeat more than maxRepeats times consecutively
func removeExcessiveRepeats(text string, maxRepeats int) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}

	result := make([]string, 0)
	i := 0

	for i < len(words) {
		currentWord := words[i]
		count := 1

		// Count consecutive repeats
		for j := i + 1; j < len(words) && words[j] == currentWord; j++ {
			count++
		}

		// Keep only up to maxRepeats occurrences
		repeatsToKeep := count
		if count > maxRepeats {
			repeatsToKeep = maxRepeats
		}

		for k := 0; k < repeatsToKeep; k++ {
			result = append(result, currentWord)
		}

		i += count
	}

	return strings.Join(result, " ")
}

// ValidateTranscriptLength checks if transcript meets minimum requirements
func ValidateTranscriptLength(transcript string, minChars int, minWords int) error {
	if len(transcript) < minChars {
		return fmt.Errorf("transcript too short: %d characters (minimum: %d)", len(transcript), minChars)
	}

	words := strings.Fields(transcript)
	if len(words) < minWords {
		return fmt.Errorf("transcript too short: %d words (minimum: %d)", len(words), minWords)
	}

	return nil
}
