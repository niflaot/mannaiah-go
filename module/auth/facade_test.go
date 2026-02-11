package auth

import (
	"testing"

	"go.uber.org/zap"
)

// TestNewFacadeValidation verifies root facade constructor delegation behavior.
func TestNewFacadeValidation(t *testing.T) {
	if _, err := New(Config{}, "development", zap.NewNop()); err == nil {
		t.Fatalf("expected New() to return validation error for empty config")
	}
}
