package store

import (
	"context"
	"errors"
	corecache "mannaiah/module/core/cache"
	"testing"
	"time"

	redisv9 "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

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

// TestNewEnablesCircuitBreaker verifies constructor wiring for Redis circuit-breaker support.
func TestNewEnablesCircuitBreaker(t *testing.T) {
	server := startMiniRedis(t)

	store, err := New(
		Config{
			URL:                     "redis://" + server.Addr() + "/0",
			CircuitBreakerEnabled:   true,
			CircuitBreakerTimeoutMS: 120000,
		},
		nil,
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	if store.breaker == nil {
		t.Fatalf("expected Redis circuit breaker to be initialized")
	}
}
