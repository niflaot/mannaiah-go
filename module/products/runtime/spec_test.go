package runtime

import "testing"

// TestOpenAPISpec verifies products OpenAPI spec shape.
func TestOpenAPISpec(t *testing.T) {
	spec := OpenAPISpec()
	if spec.Paths.Value("/products") == nil {
		t.Fatalf("expected /products path spec")
	}
	if spec.Paths.Value("/products/{id}") == nil {
		t.Fatalf("expected /products/{id} path spec")
	}
	if spec.Paths.Value("/storefront/navigation") == nil {
		t.Fatalf("expected /storefront/navigation path spec")
	}
	if spec.Paths.Value("/variations") == nil {
		t.Fatalf("expected /variations path spec")
	}
	if spec.Paths.Value("/variations/{id}") == nil {
		t.Fatalf("expected /variations/{id} path spec")
	}
	if spec.Components == nil || spec.Components.Schemas["CreateProductDto"] == nil {
		t.Fatalf("expected CreateProductDto schema")
	}
	if spec.Components == nil || spec.Components.Schemas["CreateVariationDto"] == nil {
		t.Fatalf("expected CreateVariationDto schema")
	}
	if spec.Components == nil || spec.Components.Schemas["StorefrontNavigation"] == nil {
		t.Fatalf("expected StorefrontNavigation schema")
	}
	if spec.Components == nil || spec.Components.SecuritySchemes[bearerSecurityScheme] == nil {
		t.Fatalf("expected bearer security scheme")
	}

	productsPath := spec.Paths.Value("/products")
	if productsPath == nil || productsPath.Get == nil || productsPath.Get.Security == nil || len(*productsPath.Get.Security) == 0 {
		t.Fatalf("expected bearer security requirements on products list operation")
	}
	if productsPath.Post == nil || productsPath.Post.Responses == nil || productsPath.Post.Responses.Status(409) == nil {
		t.Fatalf("expected conflict response on products create operation")
	}

	variationsPath := spec.Paths.Value("/variations")
	if variationsPath == nil || variationsPath.Get == nil || variationsPath.Get.Security == nil || len(*variationsPath.Get.Security) == 0 {
		t.Fatalf("expected bearer security requirements on variations list operation")
	}

	storefrontPath := spec.Paths.Value("/storefront/navigation")
	if storefrontPath == nil || storefrontPath.Get == nil || storefrontPath.Get.Responses == nil || storefrontPath.Get.Responses.Status(200) == nil {
		t.Fatalf("expected storefront navigation response schema")
	}
}
