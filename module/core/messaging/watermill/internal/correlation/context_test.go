package correlation

import (
	"context"
	"testing"
)

// TestWithContextAndFromContext verifies correlation storage and retrieval.
func TestWithContextAndFromContext(t *testing.T) {
	ctx := WithContext(context.Background(), "corr-1")
	value, ok := FromContext(ctx)
	if !ok {
		t.Fatalf("expected correlation id to be present")
	}
	if value != "corr-1" {
		t.Fatalf("correlation id = %q, want %q", value, "corr-1")
	}
}

// TestWithContextNilContext verifies nil context fallback behavior.
func TestWithContextNilContext(t *testing.T) {
	ctx := WithContext(nil, "corr-2")
	value, ok := FromContext(ctx)
	if !ok {
		t.Fatalf("expected correlation id to be present")
	}
	if value != "corr-2" {
		t.Fatalf("correlation id = %q, want %q", value, "corr-2")
	}
}

// TestFromContextMissing verifies empty extraction on missing correlation values.
func TestFromContextMissing(t *testing.T) {
	value, ok := FromContext(context.Background())
	if ok {
		t.Fatalf("expected missing correlation id")
	}
	if value != "" {
		t.Fatalf("expected empty correlation id, got %q", value)
	}
}
