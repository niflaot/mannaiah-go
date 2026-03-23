package runtime

import "testing"

// TestOpenAPISpec verifies shipping OpenAPI path coverage.
func TestOpenAPISpec(t *testing.T) {
	spec := OpenAPISpec()
	if spec == nil || spec.Paths == nil {
		t.Fatalf("OpenAPISpec() returned nil")
	}
	if spec.Paths.Find("/shipping/marks") == nil {
		t.Fatalf("missing /shipping/marks path")
	}
	if spec.Paths.Find("/shipping/batches") == nil {
		t.Fatalf("missing /shipping/batches path")
	}
	if spec.Paths.Find("/shipping/tracking/{trackingNumber}") == nil {
		t.Fatalf("missing /shipping/tracking/{trackingNumber} path")
	}
}
