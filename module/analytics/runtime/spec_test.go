package runtime

import "testing"

// TestOpenAPISpec verifies analytics OpenAPI contents.
func TestOpenAPISpec(t *testing.T) {
	spec := OpenAPISpec()
	if spec == nil || spec.Paths == nil {
		t.Fatalf("OpenAPISpec() returned nil")
	}
	if spec.Paths.Find("/analytics/status") == nil {
		t.Fatalf("missing /analytics/status path")
	}
}
