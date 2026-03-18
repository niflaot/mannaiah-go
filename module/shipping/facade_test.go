package shipping

import "testing"

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
