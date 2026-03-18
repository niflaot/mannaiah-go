package runtime

import "github.com/getkin/kin-openapi/openapi3"

const (
	// tagsTag defines the OpenAPI tag used by tag endpoints.
	tagsTag = "tags"
)

// tagsPathItem returns OpenAPI path operations for the tag collection endpoint.
func tagsPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{
		Get: listTagsOperation(),
	}
}

// tagByNamePathItem returns OpenAPI path operations for the tag-by-name endpoint.
func tagByNamePathItem() *openapi3.PathItem {
	return &openapi3.PathItem{
		Delete: deleteTagOperation(),
	}
}

// tagCorrelationsPathItem returns OpenAPI path operations for the correlation collection endpoint.
func tagCorrelationsPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{
		Get:  listCorrelationsOperation(),
		Post: createCorrelationOperation(),
	}
}

// tagCorrelationsBySourcePathItem returns OpenAPI path operations for source-filtered correlations.
func tagCorrelationsBySourcePathItem() *openapi3.PathItem {
	return &openapi3.PathItem{
		Get: listCorrelationsBySourceOperation(),
	}
}

// tagCorrelationByIDPathItem returns OpenAPI path operations for ID-scoped correlation endpoints.
func tagCorrelationByIDPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{
		Patch:  updateCorrelationOperation(),
		Delete: deleteCorrelationOperation(),
	}
}

// listTagsOperation defines GET /tags.
func listTagsOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "TagsController_list",
		Summary:     "List all active tags",
		Description: "Returns all non-deleted product taxonomy tags ordered by name. Requires products:read.",
		Tags:        []string{tagsTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponseBodyRef("List of active tags.", "#/components/schemas/TagListResponse")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Requires products:read.")),
		),
	}
}

// deleteTagOperation defines DELETE /tags/{name}.
func deleteTagOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "TagsController_remove",
		Summary:     "Soft-delete a tag by name",
		Description: "Soft-deletes the tag and clears it from all product_tags rows. Requires marketing:manage.",
		Tags:        []string{tagsTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathParameter("name", "Tag name to delete.", openapi3.NewStringSchema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponseBodyRef("Tag soft-deleted.", "#/components/schemas/DeleteResponse")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Requires marketing:manage.")),
			openapi3.WithStatus(404, responseWithDescription("Tag not found.")),
		),
	}
}

// listCorrelationsOperation defines GET /tags/correlations.
func listCorrelationsOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "TagCorrelationsController_list",
		Summary:     "List all tag correlations",
		Description: "Returns all tag cross-sell correlation pairs ordered by source tag. Requires marketing:manage.",
		Tags:        []string{tagsTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponseBodyRef("List of tag correlations.", "#/components/schemas/TagCorrelationListResponse")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Requires marketing:manage.")),
		),
	}
}

// listCorrelationsBySourceOperation defines GET /tags/correlations/source/{tag}.
func listCorrelationsBySourceOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "TagCorrelationsController_listBySource",
		Summary:     "List correlations by source tag",
		Description: "Returns all correlations where source_tag matches the given tag name. Requires marketing:manage.",
		Tags:        []string{tagsTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathParameter("tag", "Source tag name.", openapi3.NewStringSchema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponseBodyRef("Correlations for the source tag.", "#/components/schemas/TagCorrelationListResponse")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Requires marketing:manage.")),
		),
	}
}

// createCorrelationOperation defines POST /tags/correlations.
func createCorrelationOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "TagCorrelationsController_create",
		Summary:     "Create a tag correlation",
		Description: "Defines a cross-sell probability between two product tags. Requires marketing:manage.",
		Tags:        []string{tagsTag},
		Security:    bearerSecurityRequirements(),
		RequestBody: jsonRequestBodyRef("#/components/schemas/CreateTagCorrelationDto"),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(201, jsonResponseBodyRef("Correlation created.", "#/components/schemas/TagCorrelation")),
			openapi3.WithStatus(400, responseWithDescription("Invalid payload.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Requires marketing:manage.")),
			openapi3.WithStatus(409, responseWithDescription("Conflict - Correlation pair already exists.")),
		),
	}
}

// updateCorrelationOperation defines PATCH /tags/correlations/{id}.
func updateCorrelationOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "TagCorrelationsController_update",
		Summary:     "Update a tag correlation",
		Description: "Updates probability and/or notes for an existing correlation. Requires marketing:manage.",
		Tags:        []string{tagsTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathParameter("id", "Correlation ID.", openapi3.NewInt64Schema()),
		},
		RequestBody: jsonRequestBodyRef("#/components/schemas/UpdateTagCorrelationDto"),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponseBodyRef("Correlation updated.", "#/components/schemas/TagCorrelation")),
			openapi3.WithStatus(400, responseWithDescription("Invalid payload.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Requires marketing:manage.")),
			openapi3.WithStatus(404, responseWithDescription("Correlation not found.")),
		),
	}
}

// deleteCorrelationOperation defines DELETE /tags/correlations/{id}.
func deleteCorrelationOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "TagCorrelationsController_remove",
		Summary:     "Delete a tag correlation",
		Description: "Permanently deletes a correlation record by ID. Requires marketing:manage.",
		Tags:        []string{tagsTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathParameter("id", "Correlation ID.", openapi3.NewInt64Schema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponseBodyRef("Correlation deleted.", "#/components/schemas/DeleteResponse")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Requires marketing:manage.")),
			openapi3.WithStatus(404, responseWithDescription("Correlation not found.")),
		),
	}
}

// tagListResponseSchema returns the response schema for a list of tags.
func tagListResponseSchema() *openapi3.Schema {
	arr := openapi3.NewArraySchema()
	arr.Items = &openapi3.SchemaRef{Ref: "#/components/schemas/Tag"}
	return openapi3.NewObjectSchema().WithProperty("data", arr)
}

// tagCorrelationListResponseSchema returns the response schema for a list of correlations.
func tagCorrelationListResponseSchema() *openapi3.Schema {
	arr := openapi3.NewArraySchema()
	arr.Items = &openapi3.SchemaRef{Ref: "#/components/schemas/TagCorrelation"}
	return openapi3.NewObjectSchema().WithProperty("data", arr)
}

// deleteResponseSchema returns the response schema for successful delete operations.
func deleteResponseSchema() *openapi3.Schema {
	s := openapi3.NewObjectSchema().WithProperty("status", openapi3.NewStringSchema())
	s.Properties["status"].Value.Example = "deleted"
	return s
}

// tagSchema returns the response schema for tag payloads.
func tagSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("id", openapi3.NewInt64Schema()).
		WithProperty("name", openapi3.NewStringSchema()).
		WithProperty("createdAt", openapi3.NewDateTimeSchema()).
		WithProperty("updatedAt", openapi3.NewDateTimeSchema()).
		WithProperty("deletedAt", openapi3.NewDateTimeSchema())
}

// tagCorrelationSchema returns the response schema for tag correlation payloads.
func tagCorrelationSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("id", openapi3.NewInt64Schema()).
		WithProperty("sourceTag", openapi3.NewStringSchema()).
		WithProperty("targetTag", openapi3.NewStringSchema()).
		WithProperty("probability", openapi3.NewFloat64Schema()).
		WithProperty("notes", openapi3.NewStringSchema()).
		WithProperty("createdAt", openapi3.NewDateTimeSchema()).
		WithProperty("updatedAt", openapi3.NewDateTimeSchema())
}

// createTagCorrelationSchema returns the request schema for correlation creation.
func createTagCorrelationSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("sourceTag", openapi3.NewStringSchema()).
		WithProperty("targetTag", openapi3.NewStringSchema()).
		WithProperty("probability", openapi3.NewFloat64Schema()).
		WithProperty("notes", openapi3.NewStringSchema()).
		WithRequired([]string{"sourceTag", "targetTag"})
}

// updateTagCorrelationSchema returns the request schema for correlation updates.
func updateTagCorrelationSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("probability", openapi3.NewFloat64Schema()).
		WithProperty("notes", openapi3.NewStringSchema())
}
