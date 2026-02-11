package woocommerce

import (
	errorspkg "errors"
	"testing"
)

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

// TestNewFacadeValidation verifies root facade constructor validation behavior.
func TestNewFacadeValidation(t *testing.T) {
	if _, err := New(Config{}, nil, nil, nil); !errorspkg.Is(err, ErrNilContactService) {
		t.Fatalf("New() error = %v, want ErrNilContactService", err)
	}
}
