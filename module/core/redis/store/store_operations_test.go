package store

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

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
