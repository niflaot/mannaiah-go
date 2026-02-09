package redis

import (
	"context"
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
	return NewWithClient(client, providedLogger, cfg.ScanCount, cfg.BatchSize)
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

// Ping verifies Redis availability for the current client.
func (s *Store) Ping(ctx context.Context) error {
	if err := s.client.Ping(ctx).Err(); err != nil {
		s.logger.Error("redis ping failed", zap.Error(err))
		return fmt.Errorf("ping redis: %w", err)
	}

	return nil
}

// Get returns a value for a key or ErrNotFound when absent.
func (s *Store) Get(ctx context.Context, key string) (string, error) {
	normalizedKey, err := normalizeRequiredKey(key)
	if err != nil {
		return "", err
	}

	value, getErr := s.client.Get(ctx, normalizedKey).Result()
	if errors.Is(getErr, redisv9.Nil) {
		return "", ErrNotFound
	}
	if getErr != nil {
		s.logger.Error("redis get failed", zap.String("key", normalizedKey), zap.Error(getErr))
		return "", fmt.Errorf("redis get key %q: %w", normalizedKey, getErr)
	}

	return value, nil
}

// Set writes a value for a key with the provided TTL.
func (s *Store) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	normalizedKey, err := normalizeRequiredKey(key)
	if err != nil {
		return err
	}

	if setErr := s.client.Set(ctx, normalizedKey, value, ttl).Err(); setErr != nil {
		s.logger.Error("redis set failed", zap.String("key", normalizedKey), zap.Error(setErr))
		return fmt.Errorf("redis set key %q: %w", normalizedKey, setErr)
	}

	return nil
}

// Delete removes a key and returns the number of deleted entries.
func (s *Store) Delete(ctx context.Context, key string) (int64, error) {
	normalizedKey, err := normalizeRequiredKey(key)
	if err != nil {
		return 0, err
	}

	deleted, deleteErr := s.client.Del(ctx, normalizedKey).Result()
	if deleteErr != nil {
		s.logger.Error("redis delete failed", zap.String("key", normalizedKey), zap.Error(deleteErr))
		return 0, fmt.Errorf("redis delete key %q: %w", normalizedKey, deleteErr)
	}

	return deleted, nil
}

// Keys returns key names matching a pattern using SCAN.
func (s *Store) Keys(ctx context.Context, pattern string) ([]string, error) {
	matcher := normalizePattern(pattern)
	collected := make([]string, 0, s.scanCount)
	var cursor uint64

	for {
		keys, next, scanErr := s.client.Scan(ctx, cursor, matcher, s.scanCount).Result()
		if scanErr != nil {
			s.logger.Error("redis scan failed", zap.String("pattern", matcher), zap.Error(scanErr))
			return nil, fmt.Errorf("redis scan pattern %q: %w", matcher, scanErr)
		}

		collected = append(collected, keys...)
		if next == 0 {
			return collected, nil
		}
		cursor = next
	}
}

// GetByPattern returns key-value pairs matching a pattern using SCAN and batched MGET.
func (s *Store) GetByPattern(ctx context.Context, pattern string) (map[string]string, error) {
	keys, err := s.Keys(ctx, pattern)
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return map[string]string{}, nil
	}

	result := make(map[string]string, len(keys))
	for start := 0; start < len(keys); start += s.batchSize {
		end := start + s.batchSize
		if end > len(keys) {
			end = len(keys)
		}

		batch := keys[start:end]
		values, mgetErr := s.client.MGet(ctx, batch...).Result()
		if mgetErr != nil {
			s.logger.Error("redis mget failed", zap.Int("batch_size", len(batch)), zap.Error(mgetErr))
			return nil, fmt.Errorf("redis mget batch: %w", mgetErr)
		}

		for index, raw := range values {
			if raw == nil {
				continue
			}

			typed, castOK := raw.(string)
			if !castOK {
				continue
			}
			result[batch[index]] = typed
		}
	}

	return result, nil
}

// Close releases the underlying Redis client resources.
func (s *Store) Close() error {
	return s.client.Close()
}

// resolveLogger returns either the provided logger or a no-op logger fallback.
func resolveLogger(providedLogger *zap.Logger) *zap.Logger {
	if providedLogger != nil {
		return providedLogger
	}

	return zap.NewNop()
}

// normalizeRequiredKey validates and normalizes key input.
func normalizeRequiredKey(key string) (string, error) {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return "", ErrEmptyKey
	}

	return trimmed, nil
}

// normalizePattern normalizes empty key matchers to a wildcard.
func normalizePattern(pattern string) string {
	trimmed := strings.TrimSpace(pattern)
	if trimmed == "" {
		return "*"
	}

	return trimmed
}

// normalizeScanCount ensures SCAN count hints are always valid.
func normalizeScanCount(scanCount int64) int64 {
	if scanCount <= 0 {
		return 200
	}

	return scanCount
}

// normalizeBatchSize ensures batched reads always use a valid size.
func normalizeBatchSize(batchSize int) int {
	if batchSize <= 0 {
		return 200
	}

	return batchSize
}
