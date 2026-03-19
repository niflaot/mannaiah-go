package runtime

import "testing"

// TestOpenAPISpec verifies segment OpenAPI contents.
func TestOpenAPISpec(t *testing.T) {
	spec := OpenAPISpec()
	if spec == nil || spec.Paths == nil {
		t.Fatalf("OpenAPISpec() returned nil")
	}
	if spec.Info == nil || spec.Info.Version != "2.1.0" {
		t.Fatalf("OpenAPISpec() version = %v, want 2.1.0", spec.Info)
	}
	if spec.Paths.Find("/segments") == nil {
		t.Fatalf("missing /segments path")
	}
	if spec.Paths.Find("/segments/{id}/count") == nil {
		t.Fatalf("missing /segments/{id}/count path")
	}
	segmentSchemaRef := spec.Components.Schemas["Segment"]
	if segmentSchemaRef == nil || segmentSchemaRef.Value == nil {
		t.Fatalf("missing Segment schema")
	}
	filtersSchemaRef := segmentSchemaRef.Value.Properties["filters"]
	if filtersSchemaRef == nil || filtersSchemaRef.Value == nil || filtersSchemaRef.Value.Items == nil || filtersSchemaRef.Value.Items.Value == nil {
		t.Fatalf("missing Segment.filters schema")
	}
	if _, ok := filtersSchemaRef.Value.Items.Value.Properties["exclude"]; !ok {
		t.Fatalf("missing Segment.filters[].exclude schema property")
	}
}
