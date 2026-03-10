package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestResolveFeedResolutionBackoffBounds verifies backoff normalization behavior.
func TestResolveFeedResolutionBackoffBounds(t *testing.T) {
	if got := resolveFeedResolutionBackoff(0, 1); got != time.Second {
		t.Fatalf("resolveFeedResolutionBackoff(0,1) = %s, want %s", got, time.Second)
	}
	if got := resolveFeedResolutionBackoff(maxFeedResolutionBackoffMS+1000, 1); got != time.Duration(maxFeedResolutionBackoffMS)*time.Millisecond {
		t.Fatalf("resolveFeedResolutionBackoff(max+1000,1) = %s, want %s", got, time.Duration(maxFeedResolutionBackoffMS)*time.Millisecond)
	}
	if got := resolveFeedResolutionBackoff(500, 3); got != 1500*time.Millisecond {
		t.Fatalf("resolveFeedResolutionBackoff(500,3) = %s, want %s", got, 1500*time.Millisecond)
	}
}

// TestResolveFeedResolutionRequestTimeoutBounds verifies timeout normalization behavior.
func TestResolveFeedResolutionRequestTimeoutBounds(t *testing.T) {
	if got := resolveFeedResolutionRequestTimeout(0); got != 5*time.Second {
		t.Fatalf("resolveFeedResolutionRequestTimeout(0) = %s, want %s", got, 5*time.Second)
	}
	if got := resolveFeedResolutionRequestTimeout(maxFeedResolutionRequestTimeoutMS + 1000); got != time.Duration(maxFeedResolutionRequestTimeoutMS)*time.Millisecond {
		t.Fatalf("resolveFeedResolutionRequestTimeout(max+1000) = %s, want %s", got, time.Duration(maxFeedResolutionRequestTimeoutMS)*time.Millisecond)
	}
	if got := resolveFeedResolutionRequestTimeout(2300); got != 2300*time.Millisecond {
		t.Fatalf("resolveFeedResolutionRequestTimeout(2300) = %s, want %s", got, 2300*time.Millisecond)
	}
}

// TestWaitWithContextCanceled verifies wait cancellation behavior.
func TestWaitWithContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := waitWithContext(ctx, 50*time.Millisecond)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("waitWithContext(canceled) error = %v, want %v", err, context.Canceled)
	}
}

// TestWaitForProductFeedResolutionEmptyFeedID verifies empty product feed ID validation behavior.
func TestWaitForProductFeedResolutionEmptyFeedID(t *testing.T) {
	svc := &ProductSyncService{cfg: Config{FeedResolutionAttempts: 1}}
	err := svc.waitForProductFeedResolution(context.Background(), " ")
	if !errors.Is(err, ErrProductFeedNotResolved) {
		t.Fatalf("waitForProductFeedResolution(empty) error = %v, want %v", err, ErrProductFeedNotResolved)
	}
}
