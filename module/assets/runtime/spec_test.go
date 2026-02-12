package runtime

import "testing"

// TestOpenAPISpec verifies asset OpenAPI spec generation behavior.
func TestOpenAPISpec(t *testing.T) {
	spec := OpenAPISpec()
	if spec == nil {
		t.Fatalf("OpenAPISpec() should not return nil")
	}
	if spec.OpenAPI != "3.0.3" {
		t.Fatalf("spec.OpenAPI = %q, want %q", spec.OpenAPI, "3.0.3")
	}
	if spec.Paths == nil {
		t.Fatalf("expected non-nil paths")
	}
	if spec.Paths.Value("/assets") == nil {
		t.Fatalf("expected /assets path")
	}
	if spec.Paths.Value("/assets/{id}") == nil {
		t.Fatalf("expected /assets/{id} path")
	}
}

// TestSpecHelpers verifies helper schema and parameter behavior.
func TestSpecHelpers(t *testing.T) {
	if bearerSecurityRequirements() == nil {
		t.Fatalf("expected bearerSecurityRequirements")
	}
	if responseWithDescription("ok") == nil {
		t.Fatalf("expected responseWithDescription")
	}
	if jsonRequestBodyRef("#/components/schemas/Asset") == nil {
		t.Fatalf("expected jsonRequestBodyRef")
	}
	if pathParameter("id", "Asset ID", nil) == nil {
		t.Fatalf("expected pathParameter")
	}
	if queryParameter("page", false, "page", nil) == nil {
		t.Fatalf("expected queryParameter")
	}
	if assetsPathItem() == nil || assetByIDPathItem() == nil {
		t.Fatalf("expected path items")
	}
	if createAssetOperation() == nil || listAssetsOperation() == nil || getAssetOperation() == nil || updateAssetOperation() == nil || deleteAssetOperation() == nil {
		t.Fatalf("expected operations")
	}
	if assetSchema() == nil || updateAssetSchema() == nil || assetPaginationMetaSchema() == nil || paginatedAssetResponseSchema() == nil {
		t.Fatalf("expected schemas")
	}
}
