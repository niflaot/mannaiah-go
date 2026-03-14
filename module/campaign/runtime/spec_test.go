package runtime

import "testing"

// TestOpenAPISpec verifies campaign OpenAPI contents.
func TestOpenAPISpec(t *testing.T) {
	spec := OpenAPISpec()
	if spec == nil || spec.Paths == nil {
		t.Fatalf("OpenAPISpec() returned nil")
	}
	if spec.Paths.Find("/campaigns") == nil {
		t.Fatalf("missing /campaigns path")
	}
}
