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
	if spec.Paths.Value("/assets/folders") == nil {
		t.Fatalf("expected /assets/folders path")
	}
	if spec.Paths.Value("/assets/folders/tree") == nil {
		t.Fatalf("expected /assets/folders/tree path")
	}
	if spec.Paths.Value("/assets/folders/{id}") == nil {
		t.Fatalf("expected /assets/folders/{id} path")
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
	if assetsPathItem() == nil || assetByIDPathItem() == nil || foldersPathItem() == nil || folderByIDPathItem() == nil {
		t.Fatalf("expected path items")
	}
	if createAssetOperation() == nil || listAssetsOperation() == nil || getAssetOperation() == nil || updateAssetOperation() == nil || deleteAssetOperation() == nil {
		t.Fatalf("expected operations")
	}
	if createFolderOperation() == nil || listFoldersOperation() == nil || listFolderTreeOperation() == nil || getFolderOperation() == nil || updateFolderOperation() == nil || deleteFolderOperation() == nil {
		t.Fatalf("expected folder operations")
	}
	if assetSchema() == nil || folderSchema() == nil || folderTreeNodeSchema() == nil || folderTreeResponseSchema() == nil || tagSchema() == nil || updateAssetSchema() == nil || createFolderSchema() == nil || updateFolderSchema() == nil || assetPaginationMetaSchema() == nil || paginatedAssetResponseSchema() == nil || paginatedFolderResponseSchema() == nil {
		t.Fatalf("expected schemas")
	}

	listFolders := listFoldersOperation()
	if listFolders == nil || len(listFolders.Parameters) == 0 {
		t.Fatalf("expected listFoldersOperation parameters")
	}
	if len(listFolders.Parameters) < 4 {
		t.Fatalf("expected parentFolderId query parameter on listFoldersOperation")
	}
}
