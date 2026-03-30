package search

import (
	"testing"
)

// TestOpenAPISpecReturnsValidDocument verifies the search OpenAPI spec.
func TestOpenAPISpecReturnsValidDocument(t *testing.T) {
	spec := OpenAPISpec()
	if spec == nil {
		t.Fatal("expected non-nil spec")
	}
	if spec.OpenAPI != "3.0.3" {
		t.Errorf("openapi = %q, want 3.0.3", spec.OpenAPI)
	}
	if spec.Info == nil || spec.Info.Version != "1.1.0" {
		t.Error("expected info.version = 1.1.0")
	}
}

// TestOpenAPISpecContainsAllEndpoints verifies all search paths are present.
func TestOpenAPISpecContainsAllEndpoints(t *testing.T) {
	spec := OpenAPISpec()
	expected := []string{
		"/search/contacts",
		"/search/orders",
		"/search/products",
		"/search/categories",
		"/search/variations",
		"/search/tags",
		"/search/shipping",
		"/search/campaigns",
		"/search/segments",
		"/search",
	}
	for _, path := range expected {
		if spec.Paths.Find(path) == nil {
			t.Errorf("missing path %q", path)
		}
	}
}

// TestOpenAPISpecContainsSchemas verifies response schemas are defined.
func TestOpenAPISpecContainsSchemas(t *testing.T) {
	spec := OpenAPISpec()
	schemas := []string{"SearchResult", "SpotlightResult", "SpotlightHit"}
	for _, name := range schemas {
		if spec.Components.Schemas[name] == nil {
			t.Errorf("missing schema %q", name)
		}
	}
}

// TestOpenAPISpecResourceEndpointHasParameters verifies parameters on resource search.
func TestOpenAPISpecResourceEndpointHasParameters(t *testing.T) {
	spec := OpenAPISpec()
	pathItem := spec.Paths.Find("/search/contacts")
	if pathItem == nil || pathItem.Get == nil {
		t.Fatal("missing /contacts/search GET operation")
	}
	if len(pathItem.Get.Parameters) < 4 {
		t.Errorf("expected at least 4 parameters, got %d", len(pathItem.Get.Parameters))
	}
}

// TestOpenAPISpecSpotlightEndpoint verifies spotlight path configuration.
func TestOpenAPISpecSpotlightEndpoint(t *testing.T) {
	spec := OpenAPISpec()
	pathItem := spec.Paths.Find("/search")
	if pathItem == nil || pathItem.Get == nil {
		t.Fatal("missing /search GET operation")
	}
	if pathItem.Get.OperationID != "Search_spotlight" {
		t.Errorf("operationID = %q, want Search_spotlight", pathItem.Get.OperationID)
	}
	if len(pathItem.Get.Parameters) != 3 {
		t.Errorf("spotlight params = %d, want 3", len(pathItem.Get.Parameters))
	}
}
