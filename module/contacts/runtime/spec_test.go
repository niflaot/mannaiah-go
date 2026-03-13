package runtime

import "testing"

// TestOpenAPISpec verifies contacts OpenAPI spec shape.
func TestOpenAPISpec(t *testing.T) {
	spec := OpenAPISpec()
	if spec.Paths.Value("/contacts") == nil {
		t.Fatalf("expected /contacts path spec")
	}
	if spec.Paths.Value("/contacts/optin") == nil {
		t.Fatalf("expected /contacts/optin path spec")
	}
	if spec.Paths.Value("/contacts/optout") == nil {
		t.Fatalf("expected /contacts/optout path spec")
	}
	if spec.Paths.Value("/contacts/{id}") == nil {
		t.Fatalf("expected /contacts/{id} path spec")
	}
	if spec.Components == nil || spec.Components.Schemas["ContactCreate"] == nil {
		t.Fatalf("expected ContactCreate schema")
	}
	if spec.Components == nil || spec.Components.Schemas["ContactConsentByEmail"] == nil {
		t.Fatalf("expected ContactConsentByEmail schema")
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
	if contactsPath.Get == nil || len(contactsPath.Get.Parameters) < 1 {
		t.Fatalf("expected list contact query parameters")
	}
	hasMetadataKey := false
	hasMetadataValue := false
	for _, parameter := range contactsPath.Get.Parameters {
		if parameter == nil || parameter.Value == nil {
			continue
		}
		if parameter.Value.Name == "metadataKey" {
			hasMetadataKey = true
		}
		if parameter.Value.Name == "metadataValue" {
			hasMetadataValue = true
		}
	}
	if !hasMetadataKey || !hasMetadataValue {
		t.Fatalf("expected metadataKey and metadataValue query parameters")
	}
	createSchema := spec.Components.Schemas["ContactCreate"].Value
	if createSchema == nil || createSchema.Properties["metadata"] == nil {
		t.Fatalf("expected metadata property on ContactCreate schema")
	}
	updateSchema := spec.Components.Schemas["ContactUpdate"].Value
	if updateSchema == nil || updateSchema.Properties["metadata"] == nil {
		t.Fatalf("expected metadata property on ContactUpdate schema")
	}

	contactByIDPath := spec.Paths.Value("/contacts/{id}")
	if contactByIDPath == nil || contactByIDPath.Patch == nil || contactByIDPath.Patch.Responses == nil || contactByIDPath.Patch.Responses.Status(409) == nil {
		t.Fatalf("expected conflict response on contacts update operation")
	}

	optInPath := spec.Paths.Value("/contacts/optin")
	if optInPath == nil || optInPath.Post == nil {
		t.Fatalf("expected opt-in endpoint operation")
	}
	if optInPath.Post.Security == nil || len(*optInPath.Post.Security) == 0 {
		t.Fatalf("expected bearer security requirements on opt-in operation")
	}
	if optInPath.Post.RequestBody == nil || optInPath.Post.RequestBody.Value == nil {
		t.Fatalf("expected opt-in request body schema")
	}
	optOutPath := spec.Paths.Value("/contacts/optout")
	if optOutPath == nil || optOutPath.Post == nil {
		t.Fatalf("expected opt-out endpoint operation")
	}
	if optOutPath.Post.Responses == nil || optOutPath.Post.Responses.Status(404) == nil {
		t.Fatalf("expected not-found response on opt-out operation")
	}
}
