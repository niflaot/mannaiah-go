package runtime

import "testing"

// TestOpenAPISpec verifies email OpenAPI contents.
func TestOpenAPISpec(t *testing.T) {
	spec := OpenAPISpec()
	if spec == nil || spec.Paths == nil {
		t.Fatalf("OpenAPISpec() returned nil")
	}
	if spec.Paths.Find("/email/send") == nil {
		t.Fatalf("missing /email/send path")
	}
	if spec.Paths.Find("/email/deliveries") == nil {
		t.Fatalf("missing /email/deliveries path")
	}
	if spec.Paths.Find("/email/deliveries/{id}") == nil {
		t.Fatalf("missing /email/deliveries/{id} path")
	}
}
