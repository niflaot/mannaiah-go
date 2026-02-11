package store

import (
	"context"
	"errors"
	"testing"

	redisv9 "github.com/redis/go-redis/v9"
)

// TestPingFailsFastWhenBreakerOpen verifies open-state fail-fast behavior for repeated ping failures.
func TestPingFailsFastWhenBreakerOpen(t *testing.T) {
	server := startMiniRedis(t)

	store, err := New(
		Config{
			URL:                            "redis://" + server.Addr() + "/0",
			CircuitBreakerEnabled:          true,
			CircuitBreakerFailureThreshold: 1,
			CircuitBreakerTimeoutMS:        120000,
			CircuitBreakerIntervalMS:       120000,
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

	hook := &failingPingCounterHook{}
	client.AddHook(hook)

	if err := store.Ping(context.Background()); err == nil {
		t.Fatalf("expected first ping failure")
	}

	err = store.Ping(context.Background())
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("second Ping() error = %v, want ErrUnavailable", err)
	}
	if hook.Count() != 1 {
		t.Fatalf("ping command count = %d, want %d due to open-state fail-fast", hook.Count(), 1)
	}
}
