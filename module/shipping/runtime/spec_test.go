package runtime

import "testing"

// TestOpenAPISpec verifies shipping OpenAPI spec shape.
func TestOpenAPISpec(t *testing.T) {
	spec := OpenAPISpec()
	if spec.Paths.Value("/shipping/quotes") == nil {
		t.Fatalf("expected /shipping/quotes path spec")
	}
	if spec.Components == nil || spec.Components.SecuritySchemes[bearerSecurityScheme] == nil {
		t.Fatalf("expected bearer security scheme")
	}
	if spec.Components.Schemas["ShippingQuoteRequest"] == nil {
		t.Fatalf("expected ShippingQuoteRequest schema")
	}
	if spec.Components.Schemas["ShippingQuoteResponse"] == nil {
		t.Fatalf("expected ShippingQuoteResponse schema")
	}

	path := spec.Paths.Value("/shipping/quotes")
	if path.Post == nil {
		t.Fatalf("expected POST operation")
	}
	if path.Post.Security == nil || len(*path.Post.Security) == 0 {
		t.Fatalf("expected bearer security requirements")
	}
	if path.Post.RequestBody == nil {
		t.Fatalf("expected request body")
	}
	if path.Post.Responses == nil || path.Post.Responses.Status(200) == nil {
		t.Fatalf("expected 200 response")
	}
	if path.Post.Responses.Status(400) == nil {
		t.Fatalf("expected 400 response")
	}
	if path.Post.Responses.Status(503) == nil {
		t.Fatalf("expected 503 response")
	}
}
