package runtime

import "testing"

// TestOpenAPISpec verifies membership OpenAPI contents.
func TestOpenAPISpec(t *testing.T) {
	spec := OpenAPISpec()
	if spec == nil || spec.Paths == nil {
		t.Fatalf("OpenAPISpec() returned nil")
	}
	if spec.Paths.Find("/membership/stamp") == nil {
		t.Fatalf("missing /membership/stamp path")
	}
}
