package runtime

import "testing"

// TestOpenAPISpec verifies analytics OpenAPI spec completeness.
func TestOpenAPISpec(t *testing.T) {
	spec := OpenAPISpec()
	if spec == nil || spec.Paths == nil {
		t.Fatalf("OpenAPISpec() returned nil")
	}

	requiredPaths := []string{
		"/analytics/status",
		"/analytics/seed",
		"/analytics/rfm/bands",
		"/analytics/rfm/bands/{dimension}",
		"/analytics/rfm/groups",
		"/analytics/rfm/groups/{id}",
		"/analytics/rfm/contacts/{contactId}/score",
		"/analytics/rfm/contacts/score-batch",
		"/analytics/rfm/refresh",
		"/analytics/affinity/contacts/{contactId}",
		"/analytics/affinity/contacts/{contactId}/tags",
		"/analytics/affinity/contacts/{contactId}/categories",
		"/analytics/affinity/contacts/{contactId}/variations",
		"/analytics/affinity/refresh",
	}
	for _, path := range requiredPaths {
		if spec.Paths.Find(path) == nil {
			t.Errorf("missing path %q in OpenAPISpec", path)
		}
	}

	requiredSchemas := []string{
		"AnalyticsStatus",
		"AnalyticsSeed",
		"RFMBandConfig",
		"RFMBandUpdateRequest",
		"RFMGroup",
		"RFMGroupRequest",
		"RFMScore",
		"RFMScoreBatchRequest",
		"TagAffinity",
		"CategoryAffinity",
		"VariationAffinity",
		"AffinityProfile",
	}
	for _, schema := range requiredSchemas {
		if _, ok := spec.Components.Schemas[schema]; !ok {
			t.Errorf("missing schema %q in OpenAPISpec components", schema)
		}
	}
}
