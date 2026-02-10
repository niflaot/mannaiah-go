package contacts

import "testing"

// TestOpenAPISpec verifies contacts OpenAPI spec shape.
func TestOpenAPISpec(t *testing.T) {
	spec := OpenAPISpec()
	if spec.Paths.Value("/contacts") == nil {
		t.Fatalf("expected /contacts path spec")
	}
	if spec.Paths.Value("/contacts/{id}") == nil {
		t.Fatalf("expected /contacts/{id} path spec")
	}
	if spec.Components == nil || spec.Components.Schemas["ContactCreate"] == nil {
		t.Fatalf("expected ContactCreate schema")
	}
}
