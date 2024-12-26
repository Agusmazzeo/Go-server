package redis_utils

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"server/src/config"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// RedisHandler encapsulates the Redis client and provides utility methods.
type RedisHandler struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisHandler initializes a new Redis handler.
func NewRedisHandler(cfg *config.Config) (*RedisHandler, error) {
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{
		Addr:      cfg.Databases.Redis.Host + ":" + cfg.Databases.Redis.Port,
		Username:  cfg.Databases.Redis.Username,
		Password:  cfg.Databases.Redis.Password, // Leave empty for no password
		DB:        cfg.Databases.Redis.Database, // Default DB index
		TLSConfig: &tls.Config{InsecureSkipVerify: true},
	})

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisHandler{
		client: client,
		ctx:    ctx,
	}, nil
}

// Set stores a key-value pair in Redis with an optional expiration.
func (r *RedisHandler) Set(key string, value interface{}, expiration time.Duration) error {
	// Serialize the value to JSON
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to serialize value: %w", err)
	}

	// Store the serialized value in Redis
	return r.client.Set(r.ctx, key, data, expiration).Err()
}

// Get retrieves and deserializes the value of a key from Redis into the provided result.
func (r *RedisHandler) Get(key string, result interface{}) error {
	data, err := r.client.Get(r.ctx, key).Result()
	if err == redis.Nil {
		return fmt.Errorf("key does not exist: %s", key)
	} else if err != nil {
		return fmt.Errorf("failed to get key: %w", err)
	}

	// Deserialize the value into the result
	if err := json.Unmarshal([]byte(data), result); err != nil {
		return fmt.Errorf("failed to deserialize value: %w", err)
	}
	return nil
}

// Delete removes a key from Redis.
func (r *RedisHandler) Delete(key string) error {
	return r.client.Del(r.ctx, key).Err()
}

// Exists checks if a key exists in Redis.
func (r *RedisHandler) Exists(key string) (bool, error) {
	count, err := r.client.Exists(r.ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check key existence: %w", err)
	}
	return count > 0, nil
}

// GenerateUUID generates a real deterministic UUID (version 5) from multiple input strings.
func GenerateUUID(inputs ...string) (string, error) {
	// Use a namespace UUID (you can create a new one or use a standard one)
	namespace := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8") // DNS namespace

	// Combine inputs into a single string
	combined := ""
	for _, input := range inputs {
		combined += input
	}

	// Generate a UUIDv5
	deterministicUUID := uuid.NewMD5(namespace, []byte(combined))

	return deterministicUUID.String(), nil
}

// Close closes the Redis client connection.
func (r *RedisHandler) Close() error {
	return r.client.Close()
}
