package runtime

import "testing"

// TestOpenAPISpec verifies WooCommerce OpenAPI spec shape.
func TestOpenAPISpec(t *testing.T) {
	spec := OpenAPISpec()
	if spec.Paths.Value("/woo/sync/contacts") == nil {
		t.Fatalf("expected /woo/sync/contacts path spec")
	}
	if spec.Paths.Value("/woo/sync/orders") == nil {
		t.Fatalf("expected /woo/sync/orders path spec")
	}
	if spec.Components == nil || spec.Components.SecuritySchemes[bearerSecurityScheme] == nil {
		t.Fatalf("expected bearer security scheme")
	}

	path := spec.Paths.Value("/woo/sync/contacts")
	if path.Post == nil {
		t.Fatalf("expected POST operation")
	}
	if path.Post.Security == nil || len(*path.Post.Security) == 0 {
		t.Fatalf("expected bearer security requirements")
	}
	if path.Post.Responses == nil || path.Post.Responses.Status(503) == nil {
		t.Fatalf("expected 503 response")
	}
	if len(path.Post.Parameters) == 0 {
		t.Fatalf("expected contact sync query parameters")
	}
	if path.Post.Responses.Status(404) == nil {
		t.Fatalf("expected 404 response for contact sync")
	}

	ordersPath := spec.Paths.Value("/woo/sync/orders")
	if ordersPath.Post == nil {
		t.Fatalf("expected POST orders operation")
	}
	if ordersPath.Post.Security == nil || len(*ordersPath.Post.Security) == 0 {
		t.Fatalf("expected bearer security requirements for orders")
	}
	if ordersPath.Post.Responses == nil || ordersPath.Post.Responses.Status(503) == nil {
		t.Fatalf("expected 503 response for orders")
	}
	if len(ordersPath.Post.Parameters) == 0 {
		t.Fatalf("expected order sync query parameters")
	}
	if ordersPath.Post.Responses.Status(404) == nil {
		t.Fatalf("expected 404 response for order sync")
	}
}
