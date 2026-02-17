package runtime

import "testing"

// TestOpenAPISpec verifies Falabella OpenAPI document behavior.
func TestOpenAPISpec(t *testing.T) {
	spec := OpenAPISpec()
	if spec == nil {
		t.Fatalf("OpenAPISpec() should not return nil")
	}
	if spec.Info == nil {
		t.Fatalf("OpenAPISpec().Info should not be nil")
	}
	if spec.Info.Title != "Falabella API" {
		t.Fatalf("title = %q, want %q", spec.Info.Title, "Falabella API")
	}
	if path := spec.Paths.Value("/falabella/brands"); path == nil || path.Get == nil {
		t.Fatalf("expected GET /falabella/brands path")
	}
	if path := spec.Paths.Value("/falabella/sync/products"); path == nil || path.Post == nil {
		t.Fatalf("expected POST /falabella/sync/products path")
	}
	if path := spec.Paths.Value("/falabella/sync/products/{id}"); path == nil || path.Post == nil {
		t.Fatalf("expected POST /falabella/sync/products/{id} path")
	}
	if spec.Components == nil || spec.Components.Schemas == nil {
		t.Fatalf("expected components schemas")
	}
	if spec.Components.Schemas["FalabellaSyncSummary"] == nil {
		t.Fatalf("expected FalabellaSyncSummary schema")
	}
}
