package livekit

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/livekit/protocol/auth"
	livekit "github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

// Client wraps LiveKit operations
type Client interface {
	CreateRoom(ctx context.Context, name string, options *CreateRoomOptions) (*RoomInfo, error)
	DeleteRoom(ctx context.Context, roomName string) error
	GenerateToken(userID, roomName, participantName string, options *TokenOptions) (string, error)
	ListParticipants(ctx context.Context, roomName string) ([]*ParticipantInfo, error)
	RemoveParticipant(ctx context.Context, roomName, identity string) error
}

// CreateRoomOptions holds options for creating a room
type CreateRoomOptions struct {
	MaxParticipants  int32
	EmptyTimeout     int32 // seconds - auto-delete if no one joins
	DepartureTimeout int32 // seconds - auto-delete after last participant leaves
	Metadata         string
}

// TokenOptions holds options for generating access token
type TokenOptions struct {
	ValidFor       time.Duration
	CanPublish     bool
	CanSubscribe   bool
	CanPublishData bool
	RoomJoin       bool
	RoomAdmin      bool
}

// RoomInfo holds room information
type RoomInfo struct {
	Name            string
	SID             string
	CreationTime    time.Time
	MaxParticipants int32
	NumParticipants int32
	Metadata        string
}

// ParticipantInfo holds participant information
type ParticipantInfo struct {
	SID      string
	Identity string
	Name     string
	Metadata string
	JoinedAt time.Time
}

// realClient is the real LiveKit client implementation
type realClient struct {
	roomClient   *lksdk.RoomServiceClient
	egressClient *lksdk.EgressClient
	apiKey       string
	apiSecret    string
	url          string
}

// NewClient creates a new LiveKit client
func NewClient(url, apiKey, apiSecret string, useMock bool) Client {
	if useMock {
		return &mockClient{
			url:       url,
			apiKey:    apiKey,
			apiSecret: apiSecret,
		}
	}

	roomClient := lksdk.NewRoomServiceClient(url, apiKey, apiSecret)
	egressClient := lksdk.NewEgressClient(url, apiKey, apiSecret)
	return &realClient{
		roomClient:   roomClient,
		egressClient: egressClient,
		apiKey:       apiKey,
		apiSecret:    apiSecret,
		url:          url,
	}
}

// CreateRoom creates a new room in LiveKit
func (c *realClient) CreateRoom(ctx context.Context, name string, options *CreateRoomOptions) (*RoomInfo, error) {
	if options == nil {
		options = &CreateRoomOptions{
			MaxParticipants:  10,
			EmptyTimeout:     300, // 5 minutes
			DepartureTimeout: 30,  // 30 seconds
		}
	}

	req := &livekit.CreateRoomRequest{
		Name:             name,
		MaxParticipants:  uint32(options.MaxParticipants),
		EmptyTimeout:     uint32(options.EmptyTimeout),
		DepartureTimeout: uint32(options.DepartureTimeout),
		Metadata:         options.Metadata,
	}

	room, err := c.roomClient.CreateRoom(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create room: %w", err)
	}

	return &RoomInfo{
		Name:            room.Name,
		SID:             room.Sid,
		CreationTime:    time.Unix(room.CreationTime, 0),
		MaxParticipants: int32(room.MaxParticipants),
		NumParticipants: int32(room.NumParticipants),
		Metadata:        room.Metadata,
	}, nil
}

// DeleteRoom deletes a room from LiveKit
func (c *realClient) DeleteRoom(ctx context.Context, roomName string) error {
	_, err := c.roomClient.DeleteRoom(ctx, &livekit.DeleteRoomRequest{
		Room: roomName,
	})
	if err != nil {
		return fmt.Errorf("failed to delete room: %w", err)
	}
	return nil
}

// RemoveParticipant removes a participant from a room
func (c *realClient) RemoveParticipant(ctx context.Context, roomName, identity string) error {
	_, err := c.roomClient.RemoveParticipant(ctx, &livekit.RoomParticipantIdentity{
		Room:     roomName,
		Identity: identity,
	})
	if err != nil {
		return fmt.Errorf("failed to remove participant: %w", err)
	}
	return nil
}

// GenerateToken generates an access token for joining a room
func (c *realClient) GenerateToken(userID, roomName, participantName string, options *TokenOptions) (string, error) {
	if options == nil {
		options = &TokenOptions{
			ValidFor:       24 * time.Hour,
			CanPublish:     true,
			CanSubscribe:   true,
			CanPublishData: true,
			RoomJoin:       true,
			RoomAdmin:      false,
		}
	}

	at := auth.NewAccessToken(c.apiKey, c.apiSecret)
	grant := &auth.VideoGrant{
		RoomJoin:       options.RoomJoin,
		Room:           roomName,
		CanPublish:     &options.CanPublish,
		CanSubscribe:   &options.CanSubscribe,
		CanPublishData: &options.CanPublishData,
	}

	if options.RoomAdmin {
		grant.RoomAdmin = true
	}

	at.AddGrant(grant).
		SetIdentity(userID).
		SetName(participantName).
		SetValidFor(options.ValidFor)

	token, err := at.ToJWT()
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return token, nil
}

// ListParticipants lists all participants in a room
func (c *realClient) ListParticipants(ctx context.Context, roomName string) ([]*ParticipantInfo, error) {
	resp, err := c.roomClient.ListParticipants(ctx, &livekit.ListParticipantsRequest{
		Room: roomName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list participants: %w", err)
	}

	participants := make([]*ParticipantInfo, 0, len(resp.Participants))
	for _, p := range resp.Participants {
		participants = append(participants, &ParticipantInfo{
			SID:      p.Sid,
			Identity: p.Identity,
			Name:     p.Name,
			Metadata: p.Metadata,
			JoinedAt: time.Unix(p.JoinedAt, 0),
		})
	}

	return participants, nil
}

// mockClient is a mock implementation for testing
type mockClient struct {
	url       string
	apiKey    string
	apiSecret string
}

// CreateRoom (mock) simulates room creation
func (m *mockClient) CreateRoom(ctx context.Context, name string, options *CreateRoomOptions) (*RoomInfo, error) {
	if options == nil {
		options = &CreateRoomOptions{
			MaxParticipants: 10,
			EmptyTimeout:    300,
		}
	}

	return &RoomInfo{
		Name:            name,
		SID:             "mock-sid-" + uuid.New().String(),
		CreationTime:    time.Now(),
		MaxParticipants: options.MaxParticipants,
		NumParticipants: 0,
		Metadata:        options.Metadata,
	}, nil
}

// DeleteRoom (mock) simulates room deletion
func (m *mockClient) DeleteRoom(ctx context.Context, roomName string) error {
	// Mock: always succeed
	return nil
}

// GenerateToken (mock) generates a mock token
func (m *mockClient) GenerateToken(userID, roomName, participantName string, options *TokenOptions) (string, error) {
	if options == nil {
		options = &TokenOptions{
			ValidFor:       24 * time.Hour,
			CanPublish:     true,
			CanSubscribe:   true,
			CanPublishData: true,
			RoomJoin:       true,
		}
	}

	// Use real auth for mock too (for consistency)
	at := auth.NewAccessToken(m.apiKey, m.apiSecret)
	grant := &auth.VideoGrant{
		RoomJoin:       options.RoomJoin,
		Room:           roomName,
		CanPublish:     &options.CanPublish,
		CanSubscribe:   &options.CanSubscribe,
		CanPublishData: &options.CanPublishData,
	}

	if options.RoomAdmin {
		grant.RoomAdmin = true
	}

	at.AddGrant(grant).
		SetIdentity(userID).
		SetName(participantName).
		SetValidFor(options.ValidFor)

	token, err := at.ToJWT()
	if err != nil {
		return "", fmt.Errorf("failed to generate mock token: %w", err)
	}

	return token, nil
}

// ListParticipants (mock) returns empty list
func (m *mockClient) ListParticipants(ctx context.Context, roomName string) ([]*ParticipantInfo, error) {
	return []*ParticipantInfo{}, nil
}

// RemoveParticipant (mock) simulates participant removal
func (m *mockClient) RemoveParticipant(ctx context.Context, roomName, identity string) error {
	// Mock: always succeed
	return nil
}

// StartRoomCompositeEgress (mock) simulates starting recording
func (m *mockClient) StartRoomCompositeEgress(ctx context.Context, roomName, outputDir string) (string, error) {
	// Mock: return fake egress ID
	return "EG_mock_" + uuid.New().String(), nil
}
