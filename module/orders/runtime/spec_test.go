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
	if spec.Paths.Value("/orders/{id}/comments") == nil {
		t.Fatalf("expected /orders/{id}/comments path spec")
	}
	if spec.Paths.Value("/orders/{id}/comments/{commentId}") == nil {
		t.Fatalf("expected /orders/{id}/comments/{commentId} path spec")
	}
	if spec.Components == nil || spec.Components.Schemas["OrderCreate"] == nil {
		t.Fatalf("expected OrderCreate schema")
	}
	if spec.Components == nil || spec.Components.Schemas["OrderUpdate"] == nil {
		t.Fatalf("expected OrderUpdate schema")
	}
	if spec.Components == nil || spec.Components.Schemas["OrderAppliedCoupon"] == nil {
		t.Fatalf("expected OrderAppliedCoupon schema")
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
	if ordersPath.Post == nil || ordersPath.Post.Responses == nil || ordersPath.Post.Responses.Status(201) == nil || ordersPath.Post.Responses.Status(201).Value == nil || ordersPath.Post.Responses.Status(201).Value.Content.Get("application/json") == nil {
		t.Fatalf("expected create response schema on orders create operation")
	}
	if ordersPath.Get == nil || ordersPath.Get.Responses == nil || ordersPath.Get.Responses.Status(200) == nil || ordersPath.Get.Responses.Status(200).Value == nil || ordersPath.Get.Responses.Status(200).Value.Content.Get("application/json") == nil {
		t.Fatalf("expected list response schema on orders list operation")
	}
	if ordersPath.Get == nil || len(ordersPath.Get.Parameters) < 1 {
		t.Fatalf("expected list order query parameters")
	}
	if spec.Components.Schemas["Order"].Value.Properties["appliedCoupons"] == nil {
		t.Fatalf("expected appliedCoupons property on Order schema")
	}
	if spec.Components.Schemas["OrderCreate"].Value.Properties["appliedCoupons"] == nil {
		t.Fatalf("expected appliedCoupons property on OrderCreate schema")
	}

	orderByIDPath := spec.Paths.Value("/orders/{id}")
	if orderByIDPath == nil || orderByIDPath.Patch == nil {
		t.Fatalf("expected patch operation on /orders/{id}")
	}

	orderCommentByIDPath := spec.Paths.Value("/orders/{id}/comments/{commentId}")
	if orderCommentByIDPath == nil || orderCommentByIDPath.Patch == nil || orderCommentByIDPath.Delete == nil {
		t.Fatalf("expected patch and delete operations on /orders/{id}/comments/{commentId}")
	}
}
