package runtime

import "testing"

// TestOpenAPISpec verifies auth OpenAPI spec generation behavior.
func TestOpenAPISpec(t *testing.T) {
	spec := OpenAPISpec()
	if spec == nil {
		t.Fatalf("OpenAPISpec() should not return nil")
	}
	if spec.OpenAPI != "3.0.3" {
		t.Fatalf("spec.OpenAPI = %q, want %q", spec.OpenAPI, "3.0.3")
	}
	if spec.Paths.Value("/check-auth") == nil {
		t.Fatalf("expected /check-auth path")
	}
}

// TestSpecHelpers verifies helper behavior.
func TestSpecHelpers(t *testing.T) {
	if bearerSecurityRequirements() == nil {
		t.Fatalf("expected bearerSecurityRequirements")
	}
	if checkAuthOperation() == nil {
		t.Fatalf("expected checkAuthOperation")
	}
}
