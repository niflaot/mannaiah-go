package runtime

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
	if contactsPath.Post == nil || contactsPath.Post.Responses == nil || contactsPath.Post.Responses.Status(409) == nil {
		t.Fatalf("expected conflict response on contacts create operation")
	}

	contactByIDPath := spec.Paths.Value("/contacts/{id}")
	if contactByIDPath == nil || contactByIDPath.Patch == nil || contactByIDPath.Patch.Responses == nil || contactByIDPath.Patch.Responses.Status(409) == nil {
		t.Fatalf("expected conflict response on contacts update operation")
	}
}
