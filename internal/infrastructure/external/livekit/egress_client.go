package livekit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// EgressClient handles LiveKit egress API requests
type EgressClient struct {
	apiKey    string
	apiSecret string
	baseURL   string
	client    *http.Client
}

// NewEgressClient creates a new egress client with custom endpoint
func NewEgressClient(apiKey, apiSecret, livekitURL string) *EgressClient {
	// Extract API endpoint from LiveKit URL
	// e.g., wss://meeting-assistant-a1g5e32y.livekit.cloud -> https://meeting-assistant-a1g5e32y.livekit.cloud
	apiEndpoint := "https://" + livekitURL[6:]
	if len(apiEndpoint) > 0 && apiEndpoint[len(apiEndpoint)-1] == '/' {
		apiEndpoint = apiEndpoint[:len(apiEndpoint)-1]
	}

	return &EgressClient{
		apiKey:    apiKey,
		apiSecret: apiSecret,
		baseURL:   apiEndpoint,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// generateAccessToken generates a JWT token for LiveKit API
func (c *EgressClient) generateAccessToken() (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iss": c.apiKey,
		"sub": c.apiKey,
		"iat": now.Unix(),
		"exp": now.Add(time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(c.apiSecret))
}

// S3Config holds S3/MinIO configuration
type S3Config struct {
	AccessKey      string `json:"access_key"`
	Secret         string `json:"secret"`
	Bucket         string `json:"bucket"`
	Endpoint       string `json:"endpoint"`
	ForcePathStyle bool   `json:"force_path_style"`
	Region         string `json:"region"`
}

// FileOutput represents file output configuration
type FileOutput struct {
	Filepath string    `json:"filepath"`
	S3       *S3Config `json:"s3,omitempty"`
}

// RoomCompositeEgressRequest represents a room composite egress request
type RoomCompositeEgressRequest struct {
	RoomName   string      `json:"room_name"`
	Layout     string      `json:"layout"` // grid, speaker, single-speaker
	FileOutput *FileOutput `json:"file_output"`
}

// EgressResponse represents the response from starting egress
type EgressResponse struct {
	EgressID string `json:"egress_id"`
}

// StartRoomCompositeEgress starts recording a room to MinIO S3
func (c *EgressClient) StartRoomCompositeEgress(
	ctx context.Context,
	roomName string,
	s3Endpoint string,
	s3AccessKey string,
	s3SecretKey string,
	s3BucketName string,
) (string, error) {

	// Create S3 config
	s3Config := &S3Config{
		AccessKey:      s3AccessKey,
		Secret:         s3SecretKey,
		Bucket:         s3BucketName,
		Endpoint:       s3Endpoint,
		ForcePathStyle: true, // Required for MinIO
		Region:         "us-east-1",
	}

	// Create file output with template
	fileOutput := &FileOutput{
		Filepath: "{room_name}-{time}.mp4",
		S3:       s3Config,
	}

	// Create request
	request := &RoomCompositeEgressRequest{
		RoomName:   roomName,
		Layout:     "grid",
		FileOutput: fileOutput,
	}

	// Send request
	egressID, err := c.sendRequest(ctx, "/api/egress/room_composite", request)
	if err != nil {
		log.Printf("[Egress] ❌ Failed to start egress for room %s: %v", roomName, err)
		return "", err
	}

	log.Printf("[Egress] ✅ Started recording for room: %s (Egress ID: %s)", roomName, egressID)
	log.Printf("[Egress]    MinIO: %s", s3Endpoint)
	log.Printf("[Egress]    Bucket: %s", s3BucketName)

	return egressID, nil
}

// StopEgress stops an ongoing egress
func (c *EgressClient) StopEgress(ctx context.Context, egressID string) error {
	stopRequest := map[string]string{
		"egress_id": egressID,
	}

	data, err := json.Marshal(stopRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/egress/stop", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Generate JWT token for authorization
	token, err := c.generateAccessToken()
	if err != nil {
		return fmt.Errorf("failed to generate access token: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to stop egress: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[Egress] ❌ Failed to stop egress %s: %s (status: %d)", egressID, string(body), resp.StatusCode)
		return fmt.Errorf("failed to stop egress: status %d", resp.StatusCode)
	}

	log.Printf("[Egress] ✅ Stopped egress: %s", egressID)
	return nil
}

// sendRequest sends a request to the egress API
func (c *EgressClient) sendRequest(ctx context.Context, endpoint string, request interface{}) (string, error) {
	data, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+endpoint, bytes.NewBuffer(data))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Generate JWT token for authorization
	token, err := c.generateAccessToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate access token: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Accept both 200 OK and 201 Created
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("egress API error (status: %d): %s", resp.StatusCode, string(body))
	}

	var response EgressResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return response.EgressID, nil
}
