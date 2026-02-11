package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	redisv9 "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Ping verifies Redis availability for the current client.
func (s *Store) Ping(ctx context.Context) error {
	err := s.executeWithBreaker(func() error {
		return s.client.Ping(ctx).Err()
	})
	if err != nil {
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

	var (
		value   string
		missing bool
	)
	err = s.executeWithBreaker(func() error {
		raw, getErr := s.client.Get(ctx, normalizedKey).Result()
		if errors.Is(getErr, redisv9.Nil) {
			missing = true
			return nil
		}
		if getErr != nil {
			return getErr
		}
		value = raw
		return nil
	})
	if err != nil {
		s.logger.Error("redis get failed", zap.String("key", normalizedKey), zap.Error(err))
		return "", fmt.Errorf("redis get key %q: %w", normalizedKey, err)
	}
	if missing {
		return "", ErrNotFound
	}

	return value, nil
}

// Set writes a value for a key with the provided TTL.
func (s *Store) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	normalizedKey, err := normalizeRequiredKey(key)
	if err != nil {
		return err
	}

	err = s.executeWithBreaker(func() error {
		return s.client.Set(ctx, normalizedKey, value, ttl).Err()
	})
	if err != nil {
		s.logger.Error("redis set failed", zap.String("key", normalizedKey), zap.Error(err))
		return fmt.Errorf("redis set key %q: %w", normalizedKey, err)
	}

	return nil
}

// Delete removes a key and returns the number of deleted entries.
func (s *Store) Delete(ctx context.Context, key string) (int64, error) {
	normalizedKey, err := normalizeRequiredKey(key)
	if err != nil {
		return 0, err
	}

	var deleted int64
	err = s.executeWithBreaker(func() error {
		raw, deleteErr := s.client.Del(ctx, normalizedKey).Result()
		if deleteErr != nil {
			return deleteErr
		}
		deleted = raw
		return nil
	})
	if err != nil {
		s.logger.Error("redis delete failed", zap.String("key", normalizedKey), zap.Error(err))
		return 0, fmt.Errorf("redis delete key %q: %w", normalizedKey, err)
	}

	return deleted, nil
}

// Keys returns key names matching a pattern using SCAN.
func (s *Store) Keys(ctx context.Context, pattern string) ([]string, error) {
	matcher := normalizePattern(pattern)
	collected := make([]string, 0, s.scanCount)
	var cursor uint64

	for {
		var (
			keys []string
			next uint64
		)
		err := s.executeWithBreaker(func() error {
			var scanErr error
			keys, next, scanErr = s.client.Scan(ctx, cursor, matcher, s.scanCount).Result()
			return scanErr
		})
		if err != nil {
			s.logger.Error("redis scan failed", zap.String("pattern", matcher), zap.Error(err))
			return nil, fmt.Errorf("redis scan pattern %q: %w", matcher, err)
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
		var values []interface{}
		err := s.executeWithBreaker(func() error {
			var mgetErr error
			values, mgetErr = s.client.MGet(ctx, batch...).Result()
			return mgetErr
		})
		if err != nil {
			s.logger.Error("redis mget failed", zap.Int("batch_size", len(batch)), zap.Error(err))
			return nil, fmt.Errorf("redis mget batch: %w", err)
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
