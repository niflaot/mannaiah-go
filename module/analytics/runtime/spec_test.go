package runtime

import (
	"strings"
	"testing"
)

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
		"/analytics/recommendations/contacts/{contactId}",
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

	path := spec.Paths.Find("/analytics/recommendations/contacts/{contactId}")
	if path == nil || path.Get == nil {
		t.Fatalf("missing GET operation for recommendations path")
	}
	queryParams := map[string]string{}
	for _, parameterRef := range path.Get.Parameters {
		if parameterRef == nil || parameterRef.Value == nil {
			continue
		}
		queryParams[parameterRef.Value.Name] = parameterRef.Value.Description
	}
	if !strings.Contains(queryParams["pinnedIds"], "<product_id>|<variation_id>") {
		t.Errorf("pinnedIds description missing scoped token format: %q", queryParams["pinnedIds"])
	}
	if !strings.Contains(queryParams["excludeIds"], "<product_id>|<variation_id>") {
		t.Errorf("excludeIds description missing scoped token format: %q", queryParams["excludeIds"])
	}
	if !strings.Contains(strings.ToLower(queryParams["categoryId"]), "slug") || !strings.Contains(strings.ToLower(queryParams["categoryId"]), "name") {
		t.Errorf("categoryId description missing slug/name fallback details: %q", queryParams["categoryId"])
	}
	if !strings.Contains(strings.ToLower(queryParams["categoryId"]), "includechildren") {
		t.Errorf("categoryId description missing includeChildren behavior: %q", queryParams["categoryId"])
	}
	requiredQueryParams := []string{
		"categoryIds",
		"excludeCategoryIds",
		"includeTags",
		"excludeTags",
		"minPrice",
		"maxPrice",
		"excludePurchased",
	}
	for _, key := range requiredQueryParams {
		if strings.TrimSpace(queryParams[key]) == "" {
			t.Errorf("missing or empty recommendation query parameter description for %q", key)
		}
	}
}
