package cache

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/johnquangdev/meeting-assistant/pkg/config"
)

// RedisClient wraps redis client
type RedisClient struct {
	client *redis.Client
}

// NewRedisClient creates a new Redis client
func NewRedisClient(cfg *config.Config) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:           cfg.GetRedisAddr(),
		Password:       cfg.Redis.Password,
		DB:             cfg.Redis.DB,
		Protocol:       2,  // Use RESP2 protocol
		IdentitySuffix: "", // Disable client tracking to avoid warnings
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Println("✅ Redis connected successfully")

	return &RedisClient{
		client: client,
	}, nil
}

// Set stores a value in Redis with expiration
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

// Get retrieves a value from Redis
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

// Delete removes a key from Redis
func (r *RedisClient) Delete(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

// Exists checks if a key exists in Redis
func (r *RedisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	return r.client.Exists(ctx, keys...).Result()
}

// SetNX sets a value only if it doesn't exist
func (r *RedisClient) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return r.client.SetNX(ctx, key, value, expiration).Result()
}

// Expire sets expiration on a key
func (r *RedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return r.client.Expire(ctx, key, expiration).Err()
}

// GetTTL returns the remaining time to live of a key
func (r *RedisClient) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	return r.client.TTL(ctx, key).Result()
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	if err := r.client.Close(); err != nil {
		return fmt.Errorf("failed to close Redis connection: %w", err)
	}
	log.Println("✅ Redis connection closed")
	return nil
}

// GetClient returns the underlying Redis client for advanced operations
func (r *RedisClient) GetClient() *redis.Client {
	return r.client
}
