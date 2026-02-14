package runtime

import "testing"

// TestOpenAPISpec verifies orders OpenAPI spec shape.
func TestOpenAPISpec(t *testing.T) {
	spec := OpenAPISpec()
	if spec.Paths.Value("/orders") == nil {
		t.Fatalf("expected /orders path spec")
	}
	if spec.Paths.Value("/orders/{id}") == nil {
		t.Fatalf("expected /orders/{id} path spec")
	}
	if spec.Paths.Value("/orders/{id}/status") == nil {
		t.Fatalf("expected /orders/{id}/status path spec")
	}
	if spec.Components == nil || spec.Components.Schemas["OrderCreate"] == nil {
		t.Fatalf("expected OrderCreate schema")
	}
	if spec.Components == nil || spec.Components.SecuritySchemes[bearerSecurityScheme] == nil {
		t.Fatalf("expected bearer security scheme")
	}

	ordersPath := spec.Paths.Value("/orders")
	if ordersPath == nil || ordersPath.Get == nil || ordersPath.Get.Security == nil || len(*ordersPath.Get.Security) == 0 {
		t.Fatalf("expected bearer security requirements on orders list operation")
	}
	if ordersPath.Post == nil || ordersPath.Post.Responses == nil || ordersPath.Post.Responses.Status(409) == nil {
		t.Fatalf("expected conflict response on orders create operation")
	}
	if ordersPath.Get == nil || len(ordersPath.Get.Parameters) < 1 {
		t.Fatalf("expected list order query parameters")
	}
}
