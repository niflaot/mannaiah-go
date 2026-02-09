package cache

import (
	"context"
	"time"
)

// Store defines abstract cache capabilities independent of concrete providers.
type Store interface {
	// Ping verifies cache availability.
	Ping(ctx context.Context) error
	// Get returns a cache value for a key.
	Get(ctx context.Context, key string) (string, error)
	// Set writes a cache value for a key with an expiration timeout.
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	// Delete removes a cache key and returns the deleted count.
	Delete(ctx context.Context, key string) (int64, error)
	// Keys returns cache keys matching a pattern.
	Keys(ctx context.Context, pattern string) ([]string, error)
	// GetByPattern returns key-value entries matching a pattern.
	GetByPattern(ctx context.Context, pattern string) (map[string]string, error)
	// Close releases provider resources.
	Close() error
}
