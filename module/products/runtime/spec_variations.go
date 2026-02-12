package runtime

import "github.com/getkin/kin-openapi/openapi3"

// variationsPathItem returns OpenAPI path operations for variation collection endpoints.
func variationsPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{
		Post: createVariationOperation(),
		Get:  listVariationsOperation(),
	}
}

// variationByIDPathItem returns OpenAPI path operations for variation ID-scoped endpoints.
func variationByIDPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{
		Get:    getVariationOperation(),
		Patch:  updateVariationOperation(),
		Delete: deleteVariationOperation(),
	}
}

// createVariationOperation defines the OpenAPI operation for variation creation.
func createVariationOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "VariationsController_create",
		Summary:     "Create a new variation",
		Tags:        []string{variationsTag},
		Security:    bearerSecurityRequirements(),
		RequestBody: jsonRequestBodyRef("#/components/schemas/CreateVariationDto"),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(201, responseWithDescription("The variation has been successfully created.")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		),
	}
}

// listVariationsOperation defines the OpenAPI operation for variation listing.
func listVariationsOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "VariationsController_findAll",
		Summary:     "Get all variations",
		Tags:        []string{variationsTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Return all variations.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		),
	}
}

// getVariationOperation defines the OpenAPI operation for variation retrieval by ID.
func getVariationOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "VariationsController_findOne",
		Summary:     "Get a variation by id",
		Tags:        []string{variationsTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathParameter("id", "Variation ID", openapi3.NewStringSchema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Return the variation.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("Variation not found.")),
		),
	}
}

// updateVariationOperation defines the OpenAPI operation for variation updates.
func updateVariationOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "VariationsController_update",
		Summary:     "Update a variation",
		Tags:        []string{variationsTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathParameter("id", "Variation ID", openapi3.NewStringSchema()),
		},
		RequestBody: jsonRequestBodyRef("#/components/schemas/UpdateVariationDto"),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("The variation has been successfully updated.")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("Variation not found.")),
		),
	}
}

// deleteVariationOperation defines the OpenAPI operation for variation deletion.
func deleteVariationOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "VariationsController_remove",
		Summary:     "Delete a variation",
		Tags:        []string{variationsTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathParameter("id", "Variation ID", openapi3.NewStringSchema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("The variation has been successfully deleted.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("Variation not found.")),
		),
	}
}

// createVariationSchema returns the request schema for variation creation payloads.
func createVariationSchema() *openapi3.Schema {
	schema := openapi3.NewObjectSchema().
		WithProperty("name", openapi3.NewStringSchema()).
		WithProperty("definition", openapi3.NewStringSchema()).
		WithProperty("value", openapi3.NewStringSchema()).
		WithRequired([]string{"name", "definition", "value"})
	schema.Properties["definition"].Value.Enum = []any{"COLOR", "SIZE", "TEXT"}
	return schema
}

// updateVariationSchema returns the request schema for variation update payloads.
func updateVariationSchema() *openapi3.Schema {
	schema := openapi3.NewObjectSchema().
		WithProperty("name", openapi3.NewStringSchema()).
		WithProperty("definition", openapi3.NewStringSchema()).
		WithProperty("value", openapi3.NewStringSchema())
	schema.Properties["definition"].Value.Enum = []any{"COLOR", "SIZE", "TEXT"}
	return schema
}

// variationSchema returns the response schema for variation payloads.
func variationSchema() *openapi3.Schema {
	schema := openapi3.NewObjectSchema().
		WithProperty("_id", openapi3.NewStringSchema()).
		WithProperty("name", openapi3.NewStringSchema()).
		WithProperty("definition", openapi3.NewStringSchema()).
		WithProperty("value", openapi3.NewStringSchema()).
		WithProperty("createdAt", openapi3.NewDateTimeSchema()).
		WithProperty("updatedAt", openapi3.NewDateTimeSchema()).
		WithProperty("isDeleted", openapi3.NewBoolSchema()).
		WithProperty("deletedAt", openapi3.NewDateTimeSchema())
	schema.Properties["definition"].Value.Enum = []any{"COLOR", "SIZE", "TEXT"}
	return schema
}
