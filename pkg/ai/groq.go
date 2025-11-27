package ai

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "time"

    "github.com/johnquangdev/meeting-assistant/pkg/config"
)

// GroqClient is a minimal client for Groq API calls used for LLM analysis
type GroqClient struct {
    apiKey  string
    baseURL string
    client  *http.Client
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

    return &GroqClient{
        apiKey:  apiKey,
        baseURL: base,
        client:  &http.Client{Timeout: 30 * time.Second},
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
    prompt := fmt.Sprintf("Analyze the following transcript and return a JSON summary:\n\n%s", transcript)

    reqBody := ChatRequest{
        Model:       "llama-3.1-70b-versatile",
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
    req.Header.Set("Authorization", g.apiKey)
    req.Header.Set("Content-Type", "application/json")

    resp, err := g.client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

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
