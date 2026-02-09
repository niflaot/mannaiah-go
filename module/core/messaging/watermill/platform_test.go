package watermill

import (
	"errors"
	"testing"

	"go.uber.org/zap"
	"mannaiah/module/core/messaging/platform"
)

// TestNewInMemoryPlatform verifies root facade platform construction.
func TestNewInMemoryPlatform(t *testing.T) {
	instance, err := NewInMemoryPlatform(platform.Config{}, zap.NewNop())
	if err != nil {
		t.Fatalf("NewInMemoryPlatform() error = %v", err)
	}
	if instance == nil {
		t.Fatalf("expected non-nil platform instance")
	}

	if err := instance.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

// TestNewPublisherAdapter verifies root facade publisher construction and errors.
func TestNewPublisherAdapter(t *testing.T) {
	_, err := NewPublisherAdapter(nil)
	if !errors.Is(err, ErrNilPublisher) {
		t.Fatalf("NewPublisherAdapter() error = %v, want ErrNilPublisher", err)
	}
}
