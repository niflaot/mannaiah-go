package runtime

import "testing"

// TestOpenAPISpecIncludesExportPaths verifies export paths are documented.
func TestOpenAPISpecIncludesExportPaths(t *testing.T) {
	spec := OpenAPISpec()
	for _, path := range []string{"/exports/contacts", "/exports/orders", "/export/orders", "/exports/reports", "/exports/search"} {
		if spec.Paths.Value(path) == nil {
			t.Fatalf("missing path %s", path)
		}
	}
	if spec.Components == nil || spec.Components.Schemas["ExportReport"] == nil {
		t.Fatalf("expected ExportReport schema")
	}
	if spec.Paths.Value("/exports/contacts").Post.Responses.Status(201).Value.Content.Get("application/json") == nil {
		t.Fatalf("expected contacts generate response schema")
	}
}
