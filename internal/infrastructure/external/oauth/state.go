package oauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"
)

// RedisStore interface for Redis operations
type RedisStore interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, keys ...string) error
}

// StateManager manages OAuth state tokens for CSRF protection using Redis
type StateManager struct {
	redis      RedisStore
	expiration time.Duration
}

// NewStateManager creates a new state manager with Redis backend
func NewStateManager(redis RedisStore) *StateManager {
	return &StateManager{
		redis:      redis,
		expiration: 15 * time.Minute, // State expires in 15 minutes
	}
}

// GenerateState generates a random state token and stores it in Redis
func (sm *StateManager) GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	state := base64.URLEncoding.EncodeToString(b)

	// Store in Redis with expiration
	ctx := context.Background()
	key := fmt.Sprintf("oauth:state:%s", state)
	if err := sm.redis.Set(ctx, key, "valid", sm.expiration); err != nil {
		return "", fmt.Errorf("failed to store state in Redis: %w", err)
	}

	return state, nil
}

// ValidateState validates a state token from Redis (one-time use)
func (sm *StateManager) ValidateState(state string) bool {
	ctx := context.Background()
	key := fmt.Sprintf("oauth:state:%s", state)

	// Check if state exists in Redis
	value, err := sm.redis.Get(ctx, key)
	if err != nil || value != "valid" {
		return false
	}

	// Delete the state immediately (one-time use)
	if err := sm.redis.Delete(ctx, key); err != nil {
		// Log error but still return true since state was valid
		return true
	}

	return true
}
