package runtime

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

// TestResolveContext verifies context resolution behavior.
func TestResolveContext(t *testing.T) {
	if resolveContext(context.Background()) == nil {
		t.Fatalf("resolveContext(background) should not be nil")
	}
	var nilCtx context.Context
	if resolveContext(nilCtx) == nil {
		t.Fatalf("resolveContext(nil) should not be nil")
	}
}

// TestResolveLogger verifies logger resolution behavior.
func TestResolveLogger(t *testing.T) {
	if resolveLogger(nil) == nil {
		t.Fatalf("resolveLogger(nil) should not be nil")
	}
	logger := zap.NewNop()
	if resolveLogger(logger) != logger {
		t.Fatalf("resolveLogger(logger) should return same instance")
	}
}

// TestResolveTimeouts verifies timeout fallback behavior.
func TestResolveTimeouts(t *testing.T) {
	if resolveValidationTimeout(0) != 3*time.Second {
		t.Fatalf("resolveValidationTimeout(0) should use fallback")
	}
	if resolveRequestTimeout(0) != 5*time.Second {
		t.Fatalf("resolveRequestTimeout(0) should use fallback")
	}
}
