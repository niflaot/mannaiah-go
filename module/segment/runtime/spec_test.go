package runtime

import "testing"

// TestOpenAPISpec verifies segment OpenAPI contents.
func TestOpenAPISpec(t *testing.T) {
	spec := OpenAPISpec()
	if spec == nil || spec.Paths == nil {
		t.Fatalf("OpenAPISpec() returned nil")
	}
	if spec.Paths.Find("/segments") == nil {
		t.Fatalf("missing /segments path")
	}
	if spec.Paths.Find("/segments/{id}/count") == nil {
		t.Fatalf("missing /segments/{id}/count path")
	}
}
