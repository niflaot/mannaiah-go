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

// TestOpenAPISpecFacade verifies root facade OpenAPI delegation behavior.
func TestOpenAPISpecFacade(t *testing.T) {
	spec := OpenAPISpec()
	if spec == nil {
		t.Fatalf("OpenAPISpec() should not return nil")
	}
	if spec.OpenAPI != "3.0.3" {
		t.Fatalf("spec.OpenAPI = %q, want %q", spec.OpenAPI, "3.0.3")
	}
}
