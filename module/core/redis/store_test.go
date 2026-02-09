package redis

import (
	"context"
	"errors"
	corecache "mannaiah/module/core/cache"
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

// TestNewWithClientRejectsNilClient verifies constructor validation for nil client instances.
func TestNewWithClientRejectsNilClient(t *testing.T) {
	_, err := NewWithClient(nil, nil, 10, 10)
	if !errors.Is(err, ErrNilClient) {
		t.Fatalf("NewWithClient() error = %v, want ErrNilClient", err)
	}
}

// TestNewRejectsInvalidURL verifies URL parsing failures are returned.
func TestNewRejectsInvalidURL(t *testing.T) {
	_, err := New(
		Config{
			URL: "not-a-valid-url",
		},
		nil,
	)
	if err == nil {
		t.Fatalf("expected invalid URL parsing to fail")
	}
}

// TestNewSupportsPasswordOverride verifies config password overrides URL auth and allows ping.
func TestNewSupportsPasswordOverride(t *testing.T) {
	server := startMiniRedis(t)
	server.RequireAuth("secret")

	store, err := New(
		Config{
			URL:      "redis://" + server.Addr() + "/0",
			Password: "secret",
		},
		nil,
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	if pingErr := store.Ping(context.Background()); pingErr != nil {
		t.Fatalf("Ping() error = %v", pingErr)
	}
}

// TestNewCacheReturnsAbstractStore verifies abstract cache construction with Redis implementation.
func TestNewCacheReturnsAbstractStore(t *testing.T) {
	server := startMiniRedis(t)

	abstractStore, err := NewCache(
		Config{
			URL: "redis://" + server.Addr() + "/0",
		},
		nil,
	)
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}
	t.Cleanup(func() {
		_ = abstractStore.Close()
	})

	var _ corecache.Store = abstractStore

	if err := abstractStore.Set(context.Background(), "a", "b", 0); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	value, err := abstractStore.Get(context.Background(), "a")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if value != "b" {
		t.Fatalf("Get() = %q, want %q", value, "b")
	}
}

// TestNewAppliesConnectionOverrides verifies explicit options are propagated to go-redis client settings.
func TestNewAppliesConnectionOverrides(t *testing.T) {
	server := startMiniRedis(t)

	store, err := New(
		Config{
			URL:            "redis://" + server.Addr() + "/0",
			Username:       "user",
			Password:       "pass",
			PoolSize:       30,
			MinIdleConns:   9,
			DialTimeoutMS:  11,
			ReadTimeoutMS:  12,
			WriteTimeoutMS: 13,
			ScanCount:      321,
			BatchSize:      123,
		},
		nil,
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	client, ok := store.client.(*redisv9.Client)
	if !ok {
		t.Fatalf("expected *redis.Client, got %T", store.client)
	}

	opts := client.Options()
	if opts.Username != "user" {
		t.Fatalf("Username = %q, want %q", opts.Username, "user")
	}
	if opts.Password != "pass" {
		t.Fatalf("Password = %q, want %q", opts.Password, "pass")
	}
	if opts.PoolSize != 30 {
		t.Fatalf("PoolSize = %d, want %d", opts.PoolSize, 30)
	}
	if opts.MinIdleConns != 9 {
		t.Fatalf("MinIdleConns = %d, want %d", opts.MinIdleConns, 9)
	}
	if opts.DialTimeout != 11*time.Millisecond {
		t.Fatalf("DialTimeout = %s, want %s", opts.DialTimeout, 11*time.Millisecond)
	}
	if opts.ReadTimeout != 12*time.Millisecond {
		t.Fatalf("ReadTimeout = %s, want %s", opts.ReadTimeout, 12*time.Millisecond)
	}
	if opts.WriteTimeout != 13*time.Millisecond {
		t.Fatalf("WriteTimeout = %s, want %s", opts.WriteTimeout, 13*time.Millisecond)
	}
	if store.scanCount != 321 {
		t.Fatalf("scanCount = %d, want %d", store.scanCount, 321)
	}
	if store.batchSize != 123 {
		t.Fatalf("batchSize = %d, want %d", store.batchSize, 123)
	}
}

// TestNewWithClientDefaults verifies fallback defaults for scan and batch values.
func TestNewWithClientDefaults(t *testing.T) {
	server := startMiniRedis(t)
	client := newRedisClient(t, server)

	store, err := NewWithClient(client, nil, 0, 0)
	if err != nil {
		t.Fatalf("NewWithClient() error = %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	if store.scanCount != 200 {
		t.Fatalf("scanCount = %d, want %d", store.scanCount, 200)
	}
	if store.batchSize != 200 {
		t.Fatalf("batchSize = %d, want %d", store.batchSize, 200)
	}
	if store.logger == nil {
		t.Fatalf("expected default logger instance")
	}
}

// TestNewWithClientUsesProvidedLogger verifies caller-provided loggers are preserved.
func TestNewWithClientUsesProvidedLogger(t *testing.T) {
	server := startMiniRedis(t)
	client := newRedisClient(t, server)
	customLogger := zap.NewNop()

	store, err := NewWithClient(client, customLogger, 10, 10)
	if err != nil {
		t.Fatalf("NewWithClient() error = %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	if store.logger != customLogger {
		t.Fatalf("expected provided logger instance to be preserved")
	}
}

// TestSetGetDeleteLifecycle verifies set, get, and delete operations for a key.
func TestSetGetDeleteLifecycle(t *testing.T) {
	store := newStore(t)

	if err := store.Set(context.Background(), "user:1", "alice", 0); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	value, err := store.Get(context.Background(), "user:1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if value != "alice" {
		t.Fatalf("Get() = %q, want %q", value, "alice")
	}

	deleted, err := store.Delete(context.Background(), "user:1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted != 1 {
		t.Fatalf("Delete() = %d, want %d", deleted, 1)
	}
}

// TestSetWithTimeoutExpires verifies TTL behavior for expiring keys.
func TestSetWithTimeoutExpires(t *testing.T) {
	server := startMiniRedis(t)
	store := newStoreWithServer(t, server, 100, 100)

	if err := store.Set(context.Background(), "session:1", "active", 2*time.Second); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	server.FastForward(3 * time.Second)
	_, err := store.Get(context.Background(), "session:1")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() error = %v, want ErrNotFound after expiration", err)
	}
}

// TestGetMissingReturnsErrNotFound verifies missing keys return the sentinel not-found error.
func TestGetMissingReturnsErrNotFound(t *testing.T) {
	store := newStore(t)

	_, err := store.Get(context.Background(), "missing:key")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() error = %v, want ErrNotFound", err)
	}
}

// TestKeyValidation verifies empty-key validation for get, set, and delete.
func TestKeyValidation(t *testing.T) {
	store := newStore(t)

	if _, err := store.Get(context.Background(), "  "); !errors.Is(err, ErrEmptyKey) {
		t.Fatalf("Get() error = %v, want ErrEmptyKey", err)
	}
	if err := store.Set(context.Background(), "  ", "value", 0); !errors.Is(err, ErrEmptyKey) {
		t.Fatalf("Set() error = %v, want ErrEmptyKey", err)
	}
	if _, err := store.Delete(context.Background(), ""); !errors.Is(err, ErrEmptyKey) {
		t.Fatalf("Delete() error = %v, want ErrEmptyKey", err)
	}
}

// TestKeysByPattern verifies pattern matching and wildcard fallback behavior.
func TestKeysByPattern(t *testing.T) {
	store := newStoreWithBatching(t, 2, 2)
	populateKeyValues(t, store,
		map[string]string{
			"user:1": "alice",
			"user:2": "bob",
			"job:1":  "worker",
		},
	)

	userKeys, err := store.Keys(context.Background(), "user:*")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}
	if len(userKeys) != 2 {
		t.Fatalf("Keys() count = %d, want %d", len(userKeys), 2)
	}

	allKeys, err := store.Keys(context.Background(), "   ")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}
	if len(allKeys) != 3 {
		t.Fatalf("Keys() all count = %d, want %d", len(allKeys), 3)
	}
}

// TestGetByPattern verifies batched retrieval for key-value pairs.
func TestGetByPattern(t *testing.T) {
	store := newStoreWithBatching(t, 1, 1)
	populateKeyValues(t, store,
		map[string]string{
			"user:1": "alice",
			"user:2": "bob",
			"user:3": "carol",
			"job:1":  "worker",
		},
	)

	userValues, err := store.GetByPattern(context.Background(), "user:*")
	if err != nil {
		t.Fatalf("GetByPattern() error = %v", err)
	}
	if len(userValues) != 3 {
		t.Fatalf("GetByPattern() count = %d, want %d", len(userValues), 3)
	}
	if userValues["user:1"] != "alice" {
		t.Fatalf("GetByPattern()[user:1] = %q, want %q", userValues["user:1"], "alice")
	}

	none, err := store.GetByPattern(context.Background(), "payments:*")
	if err != nil {
		t.Fatalf("GetByPattern() error = %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("GetByPattern() expected no entries, got %d", len(none))
	}
}

// TestOperationsReturnErrorsWhenServerUnavailable verifies wrapped operation errors after disconnect.
func TestOperationsReturnErrorsWhenServerUnavailable(t *testing.T) {
	server := startMiniRedis(t)
	store := newStoreWithServer(t, server, 10, 10)

	server.Close()

	if err := store.Ping(context.Background()); err == nil || !strings.Contains(err.Error(), "ping redis") {
		t.Fatalf("Ping() error = %v, expected wrapped ping error", err)
	}
	if err := store.Set(context.Background(), "k", "v", 0); err == nil || !strings.Contains(err.Error(), "redis set key") {
		t.Fatalf("Set() error = %v, expected wrapped set error", err)
	}
	if _, err := store.Get(context.Background(), "k"); err == nil || !strings.Contains(err.Error(), "redis get key") {
		t.Fatalf("Get() error = %v, expected wrapped get error", err)
	}
	if _, err := store.Delete(context.Background(), "k"); err == nil || !strings.Contains(err.Error(), "redis delete key") {
		t.Fatalf("Delete() error = %v, expected wrapped delete error", err)
	}
	if _, err := store.Keys(context.Background(), "k*"); err == nil || !strings.Contains(err.Error(), "redis scan pattern") {
		t.Fatalf("Keys() error = %v, expected wrapped scan error", err)
	}
	if _, err := store.GetByPattern(context.Background(), "k*"); err == nil || !strings.Contains(err.Error(), "redis scan pattern") {
		t.Fatalf("GetByPattern() error = %v, expected wrapped pattern error", err)
	}
}

// TestGetByPatternReturnsMGetError verifies mget failures are wrapped and returned.
func TestGetByPatternReturnsMGetError(t *testing.T) {
	server := startMiniRedis(t)
	store := newStoreWithServer(t, server, 10, 1)
	populateKeyValues(t, store, map[string]string{"user:1": "alice"})

	store.client.AddHook(failingMGetHook{})

	_, err := store.GetByPattern(context.Background(), "user:*")
	if err == nil || !strings.Contains(err.Error(), "redis mget batch") {
		t.Fatalf("GetByPattern() error = %v, expected wrapped mget error", err)
	}
}

// TestGetByPatternSkipsNonStringValues verifies non-string MGET results are ignored.
func TestGetByPatternSkipsNonStringValues(t *testing.T) {
	server := startMiniRedis(t)
	store := newStoreWithServer(t, server, 10, 10)
	populateKeyValues(t, store, map[string]string{"user:1": "alice"})
	store.client.AddHook(nonStringMGetHook{})

	values, err := store.GetByPattern(context.Background(), "user:*")
	if err != nil {
		t.Fatalf("GetByPattern() error = %v", err)
	}
	if len(values) != 0 {
		t.Fatalf("expected no values when MGET payloads are non-strings, got %d", len(values))
	}
}

// TestClose verifies store close delegates to the underlying client.
func TestClose(t *testing.T) {
	store := newStore(t)
	if err := store.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
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
