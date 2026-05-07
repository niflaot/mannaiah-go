package runtime

import "testing"

// TestOpenAPISpec verifies Shopify OpenAPI surface completeness.
func TestOpenAPISpec(t *testing.T) {
	spec := OpenAPISpec()
	if spec == nil {
		t.Fatal("OpenAPISpec() returned nil")
	}
	if spec.OpenAPI != "3.0.3" {
		t.Fatalf("openapi = %q, want 3.0.3", spec.OpenAPI)
	}
	if spec.Components == nil || spec.Components.SecuritySchemes[bearerSecurityScheme] == nil || spec.Components.SecuritySchemes[sessionBearerSecurityScheme] == nil {
		t.Fatal("expected bearer and session bearer security schemes")
	}

	expectedPaths := []string{
		"/shopify/app",
		"/shopify/oauth/install",
		"/shopify/oauth/callback",
		"/shopify/sync/contacts",
		"/shopify/sync/orders",
		"/shopify/webhooks",
		"/shopify/ext/orders/{shopifyOrderId}",
		"/shopify/ext/contacts/{shopifyCustomerId}",
	}
	for _, path := range expectedPaths {
		if spec.Paths.Find(path) == nil {
			t.Errorf("missing path %q", path)
		}
	}

	expectedSchemas := []string{
		"ShopifyManualSyncRequest",
		"ShopifyContactSyncSummary",
		"ShopifyOrderSyncSummary",
		"ShopifyOAuthCallbackResponse",
		"ShopifyWebhookPayload",
		"ShopifyExtensionOrderSummary",
		"ShopifyExtensionContactSummary",
	}
	for _, name := range expectedSchemas {
		if spec.Components.Schemas[name] == nil {
			t.Errorf("missing schema %q", name)
		}
	}
}
