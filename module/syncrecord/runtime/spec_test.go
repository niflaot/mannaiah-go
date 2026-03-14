package runtime

import "testing"

// TestOpenAPISpec verifies sync record OpenAPI contents.
func TestOpenAPISpec(t *testing.T) {
	spec := OpenAPISpec()
	if spec == nil || spec.Paths == nil {
		t.Fatalf("OpenAPISpec() returned nil")
	}
	if spec.Paths.Find("/syncrecord/runs") == nil {
		t.Fatalf("missing /syncrecord/runs path")
	}
	if spec.Paths.Find("/syncrecord/stats") == nil {
		t.Fatalf("missing /syncrecord/stats path")
	}
}
