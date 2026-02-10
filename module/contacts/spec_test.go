package contacts

import "testing"

// TestOpenAPISpec verifies contacts OpenAPI spec shape.
func TestOpenAPISpec(t *testing.T) {
	spec := OpenAPISpec()
	if spec.Paths.Value("/contacts") == nil {
		t.Fatalf("expected /contacts path spec")
	}
	if spec.Paths.Value("/contacts/{id}") == nil {
		t.Fatalf("expected /contacts/{id} path spec")
	}
	if spec.Components == nil || spec.Components.Schemas["ContactCreate"] == nil {
		t.Fatalf("expected ContactCreate schema")
	}
	if spec.Components == nil || spec.Components.SecuritySchemes[bearerSecurityScheme] == nil {
		t.Fatalf("expected bearer security scheme")
	}

	contactsPath := spec.Paths.Value("/contacts")
	if contactsPath == nil || contactsPath.Get == nil || contactsPath.Get.Security == nil || len(*contactsPath.Get.Security) == 0 {
		t.Fatalf("expected bearer security requirements on contacts list operation")
	}
}
