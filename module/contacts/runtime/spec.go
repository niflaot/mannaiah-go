package runtime

import "github.com/getkin/kin-openapi/openapi3"

const (
	// contactsTag defines the OpenAPI tag used by contact endpoints.
	contactsTag = "contacts"
	// bearerSecurityScheme defines the OpenAPI security scheme key used for bearer auth.
	bearerSecurityScheme = "bearer"
)

// OpenAPISpec returns contact-module OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	components := openapi3.NewComponents()
	components.Schemas = openapi3.Schemas{
		"ContactCreate":         &openapi3.SchemaRef{Value: contactCreateSchema()},
		"ContactUpdate":         &openapi3.SchemaRef{Value: contactUpdateSchema()},
		"ContactConsentByEmail": &openapi3.SchemaRef{Value: contactConsentByEmailSchema()},
	}
	components.SecuritySchemes = openapi3.SecuritySchemes{
		bearerSecurityScheme: &openapi3.SecuritySchemeRef{
			Value: openapi3.NewJWTSecurityScheme(),
		},
	}

	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:   "Contacts API",
			Version: "1.0.0",
		},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/contacts", contactsPathItem()),
			openapi3.WithPath("/contacts/optin", contactOptInPathItem()),
			openapi3.WithPath("/contacts/optout", contactOptOutPathItem()),
			openapi3.WithPath("/contacts/{id}", contactByIDPathItem()),
		),
		Components: &components,
		Tags: openapi3.Tags{
			&openapi3.Tag{Name: contactsTag},
		},
	}
}

// contactOptInPathItem returns OpenAPI path operations for by-email opt-in endpoints.
func contactOptInPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{
		Post: optInByEmailOperation(),
	}
}

// contactOptOutPathItem returns OpenAPI path operations for by-email opt-out endpoints.
func contactOptOutPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{
		Post: optOutByEmailOperation(),
	}
}

// contactsPathItem returns OpenAPI path operations for collection endpoints.
func contactsPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{
		Post: createContactOperation(),
		Get:  listContactsOperation(),
	}
}

// contactByIDPathItem returns OpenAPI path operations for ID-scoped endpoints.
func contactByIDPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{
		Get:    getContactOperation(),
		Patch:  updateContactOperation(),
		Delete: deleteContactOperation(),
	}
}

// createContactOperation defines the OpenAPI operation for contact creation.
func createContactOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ContactsController_create",
		Summary:     "Create a new contact",
		Tags:        []string{contactsTag},
		Security:    bearerSecurityRequirements(),
		RequestBody: jsonRequestBodyRef("#/components/schemas/ContactCreate"),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(201, responseWithDescription("The contact has been successfully created.")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(409, responseWithDescription("Conflict - Email or document already exists.")),
		),
	}
}

// listContactsOperation defines the OpenAPI operation for contact listing.
func listContactsOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ContactsController_findAll",
		Summary:     "Get contacts with pagination and filtering",
		Tags:        []string{contactsTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			queryParameter("page", "Page number (default: 1)", integerSchema()),
			queryParameter("limit", "Items per page (default: 10)", integerSchema()),
			queryParameter("orderBy", "Field to order by (e.g., createdAt, legalName)", openapi3.NewStringSchema()),
			queryParameter("orderDir", "Order direction (asc or desc)", orderDirectionSchema()),
			queryParameter("email", "Filter by email", openapi3.NewStringSchema()),
			queryParameter("metadataKey", "Filter by metadata key", openapi3.NewStringSchema()),
			queryParameter("metadataValue", "Filter by metadata value", openapi3.NewStringSchema()),
			queryParameter("excludeIds", "Comma-separated list of IDs to exclude", openapi3.NewStringSchema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Return paginated contacts.")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		),
	}
}

// getContactOperation defines the OpenAPI operation for contact retrieval by ID.
func getContactOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ContactsController_findOne",
		Summary:     "Get a contact by id",
		Tags:        []string{contactsTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathParameter("id", "Contact ID", openapi3.NewStringSchema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Return the contact.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("Contact not found.")),
		),
	}
}

// updateContactOperation defines the OpenAPI operation for contact updates.
func updateContactOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ContactsController_update",
		Summary:     "Update a contact",
		Tags:        []string{contactsTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathParameter("id", "Contact ID", openapi3.NewStringSchema()),
		},
		RequestBody: jsonRequestBodyRef("#/components/schemas/ContactUpdate"),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("The contact has been successfully updated.")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("Contact not found.")),
			openapi3.WithStatus(409, responseWithDescription("Conflict - Email or document already exists.")),
		),
	}
}

// deleteContactOperation defines the OpenAPI operation for contact deletion.
func deleteContactOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ContactsController_remove",
		Summary:     "Delete a contact",
		Tags:        []string{contactsTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathParameter("id", "Contact ID", openapi3.NewStringSchema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("The contact has been successfully deleted.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("Contact not found.")),
		),
	}
}

// optInByEmailOperation defines the OpenAPI operation for contact opt-in updates by email.
func optInByEmailOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ContactsController_optInByEmail",
		Summary:     "Opt in a contact by email",
		Tags:        []string{contactsTag},
		Security:    bearerSecurityRequirements(),
		RequestBody: jsonRequestBodyRef("#/components/schemas/ContactConsentByEmail"),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("The contact has been successfully opted in.")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("Contact not found.")),
		),
	}
}

// optOutByEmailOperation defines the OpenAPI operation for contact opt-out updates by email.
func optOutByEmailOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ContactsController_optOutByEmail",
		Summary:     "Opt out a contact by email",
		Tags:        []string{contactsTag},
		Security:    bearerSecurityRequirements(),
		RequestBody: jsonRequestBodyRef("#/components/schemas/ContactConsentByEmail"),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("The contact has been successfully opted out.")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("Contact not found.")),
		),
	}
}

// bearerSecurityRequirements builds bearer-auth operation security requirements.
func bearerSecurityRequirements() *openapi3.SecurityRequirements {
	return openapi3.NewSecurityRequirements().With(openapi3.NewSecurityRequirement().Authenticate(bearerSecurityScheme))
}

// responseWithDescription builds an OpenAPI response from a plain description.
func responseWithDescription(description string) *openapi3.ResponseRef {
	return &openapi3.ResponseRef{
		Value: openapi3.NewResponse().WithDescription(description),
	}
}

// jsonRequestBodyRef builds a required JSON request body referencing a component schema.
func jsonRequestBodyRef(schemaRef string) *openapi3.RequestBodyRef {
	return &openapi3.RequestBodyRef{
		Value: openapi3.NewRequestBody().
			WithRequired(true).
			WithContent(openapi3.Content{
				"application/json": &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{Ref: schemaRef},
				},
			}),
	}
}

// queryParameter builds an optional query-parameter OpenAPI definition.
func queryParameter(name, description string, schema *openapi3.Schema) *openapi3.ParameterRef {
	return &openapi3.ParameterRef{
		Value: openapi3.NewQueryParameter(name).
			WithDescription(description).
			WithSchema(schema),
	}
}

// pathParameter builds a required path-parameter OpenAPI definition.
func pathParameter(name, description string, schema *openapi3.Schema) *openapi3.ParameterRef {
	return &openapi3.ParameterRef{
		Value: openapi3.NewPathParameter(name).
			WithDescription(description).
			WithSchema(schema),
	}
}

// integerSchema builds an integer schema for pagination query params.
func integerSchema() *openapi3.Schema {
	return openapi3.NewIntegerSchema()
}

// orderDirectionSchema builds the OpenAPI enum schema for list ordering direction.
func orderDirectionSchema() *openapi3.Schema {
	schema := openapi3.NewStringSchema()
	schema.Enum = []any{"asc", "desc"}
	return schema
}

// contactCreateSchema returns the request schema for contact creation payloads.
func contactCreateSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("documentType", documentTypeSchema()).
		WithProperty("documentNumber", openapi3.NewStringSchema()).
		WithProperty("legalName", openapi3.NewStringSchema()).
		WithProperty("firstName", openapi3.NewStringSchema()).
		WithProperty("lastName", openapi3.NewStringSchema()).
		WithProperty("email", openapi3.NewStringSchema()).
		WithProperty("phone", openapi3.NewStringSchema()).
		WithProperty("address", openapi3.NewStringSchema()).
		WithProperty("addressExtra", openapi3.NewStringSchema()).
		WithProperty("cityCode", openapi3.NewStringSchema()).
		WithProperty("metadata", metadataSchema()).
		WithRequired([]string{"email"})
}

// contactUpdateSchema returns the request schema for contact update payloads.
func contactUpdateSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("documentType", documentTypeSchema()).
		WithProperty("documentNumber", openapi3.NewStringSchema()).
		WithProperty("legalName", openapi3.NewStringSchema()).
		WithProperty("firstName", openapi3.NewStringSchema()).
		WithProperty("lastName", openapi3.NewStringSchema()).
		WithProperty("email", openapi3.NewStringSchema()).
		WithProperty("phone", openapi3.NewStringSchema()).
		WithProperty("address", openapi3.NewStringSchema()).
		WithProperty("addressExtra", openapi3.NewStringSchema()).
		WithProperty("cityCode", openapi3.NewStringSchema()).
		WithProperty("metadata", metadataSchema())
}

// contactConsentByEmailSchema returns the request schema for by-email consent update payloads.
func contactConsentByEmailSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("email", openapi3.NewStringSchema()).
		WithRequired([]string{"email"})
}

// documentTypeSchema returns the document-type enum schema used by contact contracts.
func documentTypeSchema() *openapi3.Schema {
	schema := openapi3.NewStringSchema()
	schema.Enum = []any{"CC", "CE", "TI", "PAS", "NIT", "OTHER"}
	return schema
}

// metadataSchema returns the metadata object schema used by contact contracts.
func metadataSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().WithAdditionalProperties(openapi3.NewStringSchema())
}
