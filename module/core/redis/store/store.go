package store

import (
	"errors"
	"fmt"
	corecache "mannaiah/module/core/cache"
	"strings"
	"time"

	redisv9 "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var (
	// ErrNotFound is returned when a key does not exist in Redis.
	ErrNotFound = errors.New("redis key not found")
	// ErrEmptyKey is returned when a key argument is blank.
	ErrEmptyKey = errors.New("redis key must not be empty")
	// ErrNilClient is returned when a nil Redis client is provided.
	ErrNilClient = errors.New("redis client must not be nil")
	// ErrUnavailable is returned when Redis operations are short-circuited by an open breaker.
	ErrUnavailable = errors.New("redis is temporarily unavailable")
)

var (
	// _ ensures Store implements the abstract provider-agnostic cache.Store contract.
	_ corecache.Store = (*Store)(nil)
)

// Store implements cache.Store backed by go-redis.
type Store struct {
	// client is the underlying Redis client implementation.
	client redisv9.UniversalClient
	// logger receives operation and failure logs.
	logger *zap.Logger
	// scanCount is the SCAN count hint for key iteration.
	scanCount int64
	// batchSize controls batched MGET read size.
	batchSize int
	// breaker guards external dependency calls and supports fail-fast behavior under outage.
	breaker Breaker
}

// New creates a Redis store from URL-based configuration and optional logger.
func New(cfg Config, providedLogger *zap.Logger) (*Store, error) {
	opts, err := redisv9.ParseURL(strings.TrimSpace(cfg.URL))
	if err != nil {
		return nil, fmt.Errorf("parse REDIS_URL %q: %w", cfg.URL, err)
	}

	if strings.TrimSpace(cfg.Username) != "" {
		opts.Username = strings.TrimSpace(cfg.Username)
	}
	if strings.TrimSpace(cfg.Password) != "" {
		opts.Password = strings.TrimSpace(cfg.Password)
	}
	if cfg.PoolSize > 0 {
		opts.PoolSize = cfg.PoolSize
	}
	if cfg.MinIdleConns > 0 {
		opts.MinIdleConns = cfg.MinIdleConns
	}
	if cfg.DialTimeoutMS > 0 {
		opts.DialTimeout = time.Duration(cfg.DialTimeoutMS) * time.Millisecond
	}
	if cfg.ReadTimeoutMS > 0 {
		opts.ReadTimeout = time.Duration(cfg.ReadTimeoutMS) * time.Millisecond
	}
	if cfg.WriteTimeoutMS > 0 {
		opts.WriteTimeout = time.Duration(cfg.WriteTimeoutMS) * time.Millisecond
	}

	client := redisv9.NewClient(opts)
	store, err := NewWithClient(client, providedLogger, cfg.ScanCount, cfg.BatchSize)
	if err != nil {
		return nil, err
	}

	if cfg.CircuitBreakerEnabled {
		breaker, breakerErr := newCircuitBreaker(cfg, store.logger)
		if breakerErr != nil {
			return nil, breakerErr
		}
		store.breaker = breaker
	}

	return store, nil
}

// NewCache creates a Redis-backed abstract cache store.
func NewCache(cfg Config, providedLogger *zap.Logger) (corecache.Store, error) {
	store, err := New(cfg, providedLogger)
	if err != nil {
		return nil, err
	}

	return store, nil
}

// NewWithClient creates a Redis store with a provided client and optional logger.
func NewWithClient(client redisv9.UniversalClient, providedLogger *zap.Logger, scanCount int64, batchSize int) (*Store, error) {
	if client == nil {
		return nil, ErrNilClient
	}

	return &Store{
		client:    client,
		logger:    resolveLogger(providedLogger),
		scanCount: normalizeScanCount(scanCount),
		batchSize: normalizeBatchSize(batchSize),
	}, nil
}
