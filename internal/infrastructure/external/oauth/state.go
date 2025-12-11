package oauth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"
)

// Store interface for state storage (in-memory)
type Store interface {
	Set(key string, value string, expiration time.Duration)
	Get(key string) (string, bool)
	Delete(key string)
}

// StateManager manages OAuth state tokens for CSRF protection using in-memory store
type StateManager struct {
	store      Store
	expiration time.Duration
}

// NewStateManager creates a new state manager with in-memory backend
func NewStateManager(store Store) *StateManager {
	return &StateManager{
		store:      store,
		expiration: 15 * time.Minute, // State expires in 15 minutes
	}
}

// GenerateState generates a random state token and stores it in-memory
func (sm *StateManager) GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	state := base64.URLEncoding.EncodeToString(b)

	// Store in memory with expiration
	key := fmt.Sprintf("oauth:state:%s", state)
	sm.store.Set(key, "valid", sm.expiration)

	return state, nil
}

// ValidateState validates a state token from in-memory store (one-time use)
func (sm *StateManager) ValidateState(state string) bool {
	key := fmt.Sprintf("oauth:state:%s", state)

	// Check if state exists and is valid
	value, exists := sm.store.Get(key)
	if !exists || value != "valid" {
		return false
	}

	// Delete the state immediately (one-time use)
	sm.store.Delete(key)

	return true
}
