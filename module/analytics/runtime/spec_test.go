package runtime

import "testing"

// TestOpenAPISpec verifies analytics OpenAPI spec completeness.
func TestOpenAPISpec(t *testing.T) {
	spec := OpenAPISpec()
	if spec == nil || spec.Paths == nil {
		t.Fatalf("OpenAPISpec() returned nil")
	}
	if spec.Paths.Len() != 0 {
		t.Fatalf("expected analytics OpenAPI surface to expose zero routes, got %d", spec.Paths.Len())
	}
	if spec.Components == nil || spec.Components.SecuritySchemes[bearerSecurityScheme] == nil {
		t.Fatalf("expected bearer security scheme")
	}
}
