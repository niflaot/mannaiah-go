package redis

import (
	corecache "mannaiah/module/core/cache"
	redisstore "mannaiah/module/core/redis/store"

	redisv9 "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var (
	// ErrNotFound is returned when a key does not exist in Redis.
	ErrNotFound = redisstore.ErrNotFound
	// ErrEmptyKey is returned when a key argument is blank.
	ErrEmptyKey = redisstore.ErrEmptyKey
	// ErrNilClient is returned when a nil Redis client is provided.
	ErrNilClient = redisstore.ErrNilClient
	// ErrUnavailable is returned when Redis operations are short-circuited by an open breaker.
	ErrUnavailable = redisstore.ErrUnavailable
)

// Store implements cache.Store backed by go-redis.
type Store = redisstore.Store

// New creates a Redis store from URL-based configuration and optional logger.
func New(cfg Config, providedLogger *zap.Logger) (*Store, error) {
	return redisstore.New(cfg, providedLogger)
}

// NewCache creates a Redis-backed abstract cache store.
func NewCache(cfg Config, providedLogger *zap.Logger) (corecache.Store, error) {
	return redisstore.NewCache(cfg, providedLogger)
}

// NewWithClient creates a Redis store with a provided client and optional logger.
func NewWithClient(client redisv9.UniversalClient, providedLogger *zap.Logger, scanCount int64, batchSize int) (*Store, error) {
	return redisstore.NewWithClient(client, providedLogger, scanCount, batchSize)
}
