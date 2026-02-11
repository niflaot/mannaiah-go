package redis

import (
	"context"
	"errors"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	redisv9 "github.com/redis/go-redis/v9"
	redisstore "mannaiah/module/core/redis/store"
)

// TestErrorAliases verifies root-package sentinel error alias behavior.
func TestErrorAliases(t *testing.T) {
	if !errors.Is(ErrNotFound, redisstore.ErrNotFound) {
		t.Fatalf("ErrNotFound should alias store.ErrNotFound")
	}
	if !errors.Is(ErrEmptyKey, redisstore.ErrEmptyKey) {
		t.Fatalf("ErrEmptyKey should alias store.ErrEmptyKey")
	}
	if !errors.Is(ErrNilClient, redisstore.ErrNilClient) {
		t.Fatalf("ErrNilClient should alias store.ErrNilClient")
	}
	if !errors.Is(ErrUnavailable, redisstore.ErrUnavailable) {
		t.Fatalf("ErrUnavailable should alias store.ErrUnavailable")
	}
}

// TestNewWithClientWrapper verifies root-package constructor forwarding behavior.
func TestNewWithClientWrapper(t *testing.T) {
	if _, err := NewWithClient(nil, nil, 10, 10); !errors.Is(err, ErrNilClient) {
		t.Fatalf("NewWithClient(nil) error = %v, want ErrNilClient", err)
	}
}

// TestNewAndNewCacheWrappers verifies root-package wrapper behavior for constructors.
func TestNewAndNewCacheWrappers(t *testing.T) {
	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	defer server.Close()

	cfg := Config{
		URL: "redis://" + server.Addr() + "/0",
	}

	store, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	if err := store.Set(context.Background(), "a", "b", 0); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	cacheStore, err := NewCache(cfg, nil)
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}
	defer func() {
		_ = cacheStore.Close()
	}()
}

// TestConfigAliasUsage verifies root-package config alias compatibility.
func TestConfigAliasUsage(t *testing.T) {
	cfg := Config{}
	cfg.URL = "redis://localhost:6379/0"
	if cfg.URL == "" {
		t.Fatalf("config alias should expose URL field")
	}
}

// TestWrapperTypeAlias verifies root-package Store alias compatibility.
func TestWrapperTypeAlias(t *testing.T) {
	client := redisv9.NewClient(&redisv9.Options{Addr: "localhost:6379", DB: 0})
	defer client.Close()

	_, _ = NewWithClient(client, nil, 10, 10)
}
