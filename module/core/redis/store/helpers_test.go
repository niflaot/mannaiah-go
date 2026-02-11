package store

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	redisv9 "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// failingMGetHook forces MGET commands to fail for branch coverage.
type failingMGetHook struct{}

// DialHook returns the wrapped dial hook unchanged.
func (h failingMGetHook) DialHook(next redisv9.DialHook) redisv9.DialHook {
	return next
}

// ProcessHook injects an error for MGET commands.
func (h failingMGetHook) ProcessHook(next redisv9.ProcessHook) redisv9.ProcessHook {
	return func(ctx context.Context, cmd redisv9.Cmder) error {
		if strings.EqualFold(cmd.Name(), "mget") {
			return errors.New("forced mget error")
		}

		return next(ctx, cmd)
	}
}

// ProcessPipelineHook returns the wrapped pipeline hook unchanged.
func (h failingMGetHook) ProcessPipelineHook(next redisv9.ProcessPipelineHook) redisv9.ProcessPipelineHook {
	return next
}

// nonStringMGetHook mutates MGET results with non-string payloads.
type nonStringMGetHook struct{}

// DialHook returns the wrapped dial hook unchanged.
func (h nonStringMGetHook) DialHook(next redisv9.DialHook) redisv9.DialHook {
	return next
}

// ProcessHook rewrites MGET values so type assertions fail.
func (h nonStringMGetHook) ProcessHook(next redisv9.ProcessHook) redisv9.ProcessHook {
	return func(ctx context.Context, cmd redisv9.Cmder) error {
		if err := next(ctx, cmd); err != nil {
			return err
		}

		if strings.EqualFold(cmd.Name(), "mget") {
			sliceCmd, ok := cmd.(*redisv9.SliceCmd)
			if ok {
				raw := sliceCmd.Val()
				converted := make([]interface{}, len(raw))
				for index := range raw {
					converted[index] = 123
				}
				sliceCmd.SetVal(converted)
			}
		}

		return nil
	}
}

// ProcessPipelineHook returns the wrapped pipeline hook unchanged.
func (h nonStringMGetHook) ProcessPipelineHook(next redisv9.ProcessPipelineHook) redisv9.ProcessPipelineHook {
	return next
}

// failingPingCounterHook forces ping failures and tracks command execution count.
type failingPingCounterHook struct {
	// count defines intercepted ping command count.
	count int
}

// DialHook returns the wrapped dial hook unchanged.
func (h *failingPingCounterHook) DialHook(next redisv9.DialHook) redisv9.DialHook {
	return next
}

// ProcessHook injects an error for ping commands and tracks execution count.
func (h *failingPingCounterHook) ProcessHook(next redisv9.ProcessHook) redisv9.ProcessHook {
	return func(ctx context.Context, cmd redisv9.Cmder) error {
		if strings.EqualFold(cmd.Name(), "ping") {
			h.count++
			return errors.New("forced ping error")
		}

		return next(ctx, cmd)
	}
}

// ProcessPipelineHook returns the wrapped pipeline hook unchanged.
func (h *failingPingCounterHook) ProcessPipelineHook(next redisv9.ProcessPipelineHook) redisv9.ProcessPipelineHook {
	return next
}

// Count reports intercepted ping execution count.
func (h *failingPingCounterHook) Count() int {
	return h.count
}

// startMiniRedis starts an isolated in-memory Redis server.
func startMiniRedis(t *testing.T) *miniredis.Miniredis {
	t.Helper()

	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	t.Cleanup(server.Close)

	return server
}

// newRedisClient creates a go-redis client for a miniredis server.
func newRedisClient(t *testing.T, server *miniredis.Miniredis) *redisv9.Client {
	t.Helper()

	return redisv9.NewClient(&redisv9.Options{
		Addr:         server.Addr(),
		DB:           0,
		DialTimeout:  50 * time.Millisecond,
		ReadTimeout:  50 * time.Millisecond,
		WriteTimeout: 50 * time.Millisecond,
	})
}

// newStore creates a store with default scan and batch behavior.
func newStore(t *testing.T) *Store {
	t.Helper()

	return newStoreWithBatching(t, 10, 10)
}

// newStoreWithBatching creates a store with explicit scan and batch configuration.
func newStoreWithBatching(t *testing.T, scanCount int64, batchSize int) *Store {
	t.Helper()

	server := startMiniRedis(t)
	return newStoreWithServer(t, server, scanCount, batchSize)
}

// newStoreWithServer creates a store bound to a specific miniredis server.
func newStoreWithServer(t *testing.T, server *miniredis.Miniredis, scanCount int64, batchSize int) *Store {
	t.Helper()

	client := newRedisClient(t, server)
	store, err := NewWithClient(client, zap.NewNop(), scanCount, batchSize)
	if err != nil {
		t.Fatalf("NewWithClient() error = %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	return store
}

// populateKeyValues seeds key-value entries in Redis for tests.
func populateKeyValues(t *testing.T, store *Store, entries map[string]string) {
	t.Helper()

	for key, value := range entries {
		if err := store.Set(context.Background(), key, value, 0); err != nil {
			t.Fatalf("Set() error for %q = %v", key, err)
		}
	}
}
