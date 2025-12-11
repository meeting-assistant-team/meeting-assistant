package cache

import (
	"sync"
	"time"
)

// MemoryStore is a simple in-memory key-value store with expiration
type MemoryStore struct {
	mu    sync.RWMutex
	items map[string]*memoryItem
}

type memoryItem struct {
	value      string
	expireTime time.Time
}

// NewMemoryStore creates a new in-memory store
func NewMemoryStore() *MemoryStore {
	store := &MemoryStore{
		items: make(map[string]*memoryItem),
	}

	// Start cleanup goroutine to remove expired items
	go store.cleanupExpired()

	return store
}

// Set stores a key-value pair with expiration
func (ms *MemoryStore) Set(key string, value string, expiration time.Duration) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.items[key] = &memoryItem{
		value:      value,
		expireTime: time.Now().Add(expiration),
	}
}

// Get retrieves a value by key (returns empty string if not found or expired)
func (ms *MemoryStore) Get(key string) (string, bool) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	item, exists := ms.items[key]
	if !exists {
		return "", false
	}

	// Check if expired
	if time.Now().After(item.expireTime) {
		return "", false
	}

	return item.value, true
}

// Delete removes a key
func (ms *MemoryStore) Delete(key string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	delete(ms.items, key)
}

// cleanupExpired periodically removes expired items
func (ms *MemoryStore) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ms.mu.Lock()
		now := time.Now()
		for key, item := range ms.items {
			if now.After(item.expireTime) {
				delete(ms.items, key)
			}
		}
		ms.mu.Unlock()
	}
}
