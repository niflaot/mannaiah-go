package runtime

import (
	"strings"
	"testing"
)

// TestOpenAPISpec verifies campaign OpenAPI contents.
func TestOpenAPISpec(t *testing.T) {
	spec := OpenAPISpec()
	if spec == nil || spec.Paths == nil {
		t.Fatalf("OpenAPISpec() returned nil")
	}
	if spec.Paths.Find("/campaigns") == nil {
		t.Fatalf("missing /campaigns path")
	}

	blockSchemaRef, ok := spec.Components.Schemas["CampaignProductBlock"]
	if !ok || blockSchemaRef == nil || blockSchemaRef.Value == nil {
		t.Fatalf("missing CampaignProductBlock schema")
	}
	pinnedSchema := blockSchemaRef.Value.Properties["pinnedProductIds"]
	if pinnedSchema == nil || pinnedSchema.Value == nil {
		t.Fatalf("missing pinnedProductIds schema property")
	}
	if !strings.Contains(pinnedSchema.Value.Description, "<product_id>|<variation_id>") {
		t.Errorf("pinnedProductIds description missing scoped token format: %q", pinnedSchema.Value.Description)
	}

	excludedSchema := blockSchemaRef.Value.Properties["excludeProductIds"]
	if excludedSchema == nil || excludedSchema.Value == nil {
		t.Fatalf("missing excludeProductIds schema property")
	}
	if !strings.Contains(excludedSchema.Value.Description, "<product_id>|<variation_id>") {
		t.Errorf("excludeProductIds description missing scoped token format: %q", excludedSchema.Value.Description)
	}

	categorySchema := blockSchemaRef.Value.Properties["categoryId"]
	if categorySchema == nil || categorySchema.Value == nil {
		t.Fatalf("missing categoryId schema property")
	}
	categoryDescription := strings.ToLower(categorySchema.Value.Description)
	if !strings.Contains(categoryDescription, "slug") || !strings.Contains(categoryDescription, "name") {
		t.Errorf("categoryId description missing slug/name fallback details: %q", categorySchema.Value.Description)
	}
	if !strings.Contains(categoryDescription, "includechildren") {
		t.Errorf("categoryId description missing includeChildren behavior: %q", categorySchema.Value.Description)
	}

	requiredProperties := []string{
		"categoryIds",
		"excludeCategoryIds",
		"includeTags",
		"excludeTags",
		"minPrice",
		"maxPrice",
		"excludePurchasedProducts",
	}
	for _, property := range requiredProperties {
		if blockSchemaRef.Value.Properties[property] == nil {
			t.Errorf("missing CampaignProductBlock property %q", property)
		}
	}
}
