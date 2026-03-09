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
	if path := spec.Paths.Value("/falabella/images/transcoded"); path == nil || path.Get == nil {
		t.Fatalf("expected GET /falabella/images/transcoded path")
	}
	if path := spec.Paths.Value("/falabella/sync/products"); path == nil || path.Post == nil {
		t.Fatalf("expected POST /falabella/sync/products path")
	}
	if path := spec.Paths.Value("/falabella/sync/products/{id}"); path == nil || path.Post == nil {
		t.Fatalf("expected POST /falabella/sync/products/{id} path")
	}
	if path := spec.Paths.Value("/falabella/sync/status/feed/{feedId}"); path == nil || path.Get == nil {
		t.Fatalf("expected GET /falabella/sync/status/feed/{feedId} path")
	}
	if path := spec.Paths.Value("/falabella/sync/status/execution/{executionId}"); path == nil || path.Get == nil {
		t.Fatalf("expected GET /falabella/sync/status/execution/{executionId} path")
	}
	if path := spec.Paths.Value("/falabella/sync/status/execution/{executionId}/feeds"); path == nil || path.Get == nil {
		t.Fatalf("expected GET /falabella/sync/status/execution/{executionId}/feeds path")
	}
	if path := spec.Paths.Value("/falabella/sync/status/product/{productId}"); path == nil || path.Get == nil {
		t.Fatalf("expected GET /falabella/sync/status/product/{productId} path")
	}
	if path := spec.Paths.Value("/falabella/sync/status/feed/{feedId}/resolve"); path == nil || path.Post == nil {
		t.Fatalf("expected POST /falabella/sync/status/feed/{feedId}/resolve path")
	}
	if spec.Components == nil || spec.Components.Schemas == nil {
		t.Fatalf("expected components schemas")
	}
	if spec.Components.Schemas["FalabellaSyncSummary"] == nil {
		t.Fatalf("expected FalabellaSyncSummary schema")
	}
	if spec.Components.Schemas["FalabellaSyncStatusEntry"] == nil {
		t.Fatalf("expected FalabellaSyncStatusEntry schema")
	}
	entrySchema := spec.Components.Schemas["FalabellaSyncStatusEntry"]
	if entrySchema == nil || entrySchema.Value == nil || entrySchema.Value.Properties["variationIds"] == nil {
		t.Fatalf("expected variationIds property in FalabellaSyncStatusEntry schema")
	}
	if entrySchema.Value.Properties["task"] == nil {
		t.Fatalf("expected task property in FalabellaSyncStatusEntry schema")
	}
	if spec.Components.Schemas["FalabellaSyncStatusExecution"] == nil {
		t.Fatalf("expected FalabellaSyncStatusExecution schema")
	}
	if spec.Components.Schemas["FalabellaResolveResult"] == nil {
		t.Fatalf("expected FalabellaResolveResult schema")
	}
	resolveSchema := spec.Components.Schemas["FalabellaResolveResult"]
	if resolveSchema == nil || resolveSchema.Value == nil || resolveSchema.Value.Properties["task"] == nil {
		t.Fatalf("expected task property in FalabellaResolveResult schema")
	}
}
