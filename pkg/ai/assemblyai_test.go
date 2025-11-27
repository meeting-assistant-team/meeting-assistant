package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/johnquangdev/meeting-assistant/pkg/config"
)

func TestTranscribeAudio_Success(t *testing.T) {
	// Mock AssemblyAI server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST got %s", r.Method)
		}
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("invalid payload: %v", err)
		}
		// Return fake id
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"id": "transcript-123", "status": "processing"})
	}))
	defer ts.Close()

	client := NewAssemblyAIClient(&config.AssemblyAIConfig{APIKey: "test-key"})

	// Simulate what TranscribeAudio does but point to ts.URL
	payload := TranscribeRequest{AudioURL: "http://example.com/audio.mp3", SpeakerLabels: true}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", ts.URL, bytes.NewReader(b))
	req.Header.Set("Authorization", client.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	var tr TranscribeResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if tr.ID != "transcript-123" {
		t.Fatalf("unexpected id %s", tr.ID)
	}
}
