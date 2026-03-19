package runtime

import "github.com/getkin/kin-openapi/openapi3"

const (
	// segmentTag defines segment OpenAPI tag values.
	segmentTag = "segments"
	// bearerSecurityScheme defines security-scheme key values.
	bearerSecurityScheme = "segment_bearer"
)

// OpenAPISpec returns segment-module OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	components := openapi3.NewComponents()
	components.SecuritySchemes = openapi3.SecuritySchemes{
		bearerSecurityScheme: &openapi3.SecuritySchemeRef{Value: openapi3.NewJWTSecurityScheme()},
	}
	components.Schemas = openapi3.Schemas{
		"Segment":             {Value: segmentSchema()},
		"SegmentListResult":   {Value: segmentListSchema()},
		"SegmentResolve":      {Value: resolveSchema()},
		"SegmentCount":        {Value: countSchema()},
		"SegmentPreviewCount": {Value: previewCountSchema()},
		"SegmentDelete":       {Value: deleteSchema()},
	}

	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info:    &openapi3.Info{Title: "Segment API", Version: "2.0.8"},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/segments/preview/count", &openapi3.PathItem{Post: previewCountOperation()}),
			openapi3.WithPath("/segments", &openapi3.PathItem{Post: createOperation(), Get: listOperation()}),
			openapi3.WithPath("/segments/{id}", &openapi3.PathItem{Get: getOperation(), Patch: updateOperation(), Delete: deleteOperation()}),
			openapi3.WithPath("/segments/{id}/resolve", &openapi3.PathItem{Post: resolveOperation()}),
			openapi3.WithPath("/segments/{id}/count", &openapi3.PathItem{Get: countOperation()}),
		),
		Components: &components,
		Tags:       openapi3.Tags{&openapi3.Tag{Name: segmentTag}},
	}
}

// createOperation builds create segment OpenAPI operations.
func createOperation() *openapi3.Operation {
	operation := baseOperation("SegmentController_create", "Create segment")
	operation.Responses = openapi3.NewResponses(
		openapi3.WithStatus(201, jsonResponse("Segment created.", "#/components/schemas/Segment")),
		openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
		openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
		openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
	)

	return operation
}

// listOperation builds list segments OpenAPI operations.
func listOperation() *openapi3.Operation {
	operation := baseOperation("SegmentController_list", "List segments")
	operation.Responses = openapi3.NewResponses(
		openapi3.WithStatus(200, jsonResponse("Segment list.", "#/components/schemas/SegmentListResult")),
		openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
		openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
	)

	return operation
}

// getOperation builds get segment OpenAPI operations.
func getOperation() *openapi3.Operation {
	operation := baseOperation("SegmentController_get", "Get segment")
	operation.Responses = openapi3.NewResponses(
		openapi3.WithStatus(200, jsonResponse("Segment.", "#/components/schemas/Segment")),
		openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
		openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		openapi3.WithStatus(404, responseWithDescription("Segment not found.")),
	)

	return operation
}

// updateOperation builds update segment OpenAPI operations.
func updateOperation() *openapi3.Operation {
	operation := baseOperation("SegmentController_update", "Update segment")
	operation.Responses = openapi3.NewResponses(
		openapi3.WithStatus(200, jsonResponse("Segment updated.", "#/components/schemas/Segment")),
		openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
		openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
		openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		openapi3.WithStatus(404, responseWithDescription("Segment not found.")),
	)

	return operation
}

// deleteOperation builds delete segment OpenAPI operations.
func deleteOperation() *openapi3.Operation {
	operation := baseOperation("SegmentController_delete", "Delete segment")
	operation.Responses = openapi3.NewResponses(
		openapi3.WithStatus(200, jsonResponse("Segment deleted.", "#/components/schemas/SegmentDelete")),
		openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
		openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		openapi3.WithStatus(404, responseWithDescription("Segment not found.")),
	)

	return operation
}

// resolveOperation builds resolve segment OpenAPI operations.
func resolveOperation() *openapi3.Operation {
	operation := baseOperation("SegmentController_resolve", "Resolve segment")
	operation.Responses = openapi3.NewResponses(
		openapi3.WithStatus(200, jsonResponse("Segment contacts resolved.", "#/components/schemas/SegmentResolve")),
		openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
		openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		openapi3.WithStatus(404, responseWithDescription("Segment not found.")),
		openapi3.WithStatus(503, responseWithDescription("Segment backend unavailable.")),
	)

	return operation
}

// countOperation builds segment-count OpenAPI operations.
func countOperation() *openapi3.Operation {
	operation := baseOperation("SegmentController_count", "Count segment contacts")
	operation.Responses = openapi3.NewResponses(
		openapi3.WithStatus(200, jsonResponse("Segment count.", "#/components/schemas/SegmentCount")),
		openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
		openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		openapi3.WithStatus(404, responseWithDescription("Segment not found.")),
		openapi3.WithStatus(503, responseWithDescription("Segment backend unavailable.")),
	)

	return operation
}

// baseOperation builds one standard segment operation.
func baseOperation(id string, summary string) *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: id,
		Summary:     summary,
		Tags:        []string{segmentTag},
		Security:    bearerSecurityRequirements(),
	}
}

// segmentSchema defines segment response schema values.
func segmentSchema() *openapi3.Schema {
	filterSchema := openapi3.NewObjectSchema().
		WithProperty("type", openapi3.NewStringSchema()).
		WithProperty("exclude", openapi3.NewBoolSchema()).
		WithProperty("value", openapi3.NewObjectSchema()).
		WithProperty("parameters", openapi3.NewObjectSchema())

	return openapi3.NewObjectSchema().
		WithProperty("id", openapi3.NewStringSchema()).
		WithProperty("name", openapi3.NewStringSchema()).
		WithProperty("slug", openapi3.NewStringSchema()).
		WithProperty("channel", openapi3.NewStringSchema()).
		WithProperty("filters", openapi3.NewArraySchema().WithItems(filterSchema)).
		WithProperty("createdAt", openapi3.NewDateTimeSchema()).
		WithProperty("updatedAt", openapi3.NewDateTimeSchema())
}

// segmentListSchema defines segment-list response schema values.
func segmentListSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("data", openapi3.NewArraySchema().WithItems(segmentSchema())).
		WithProperty("page", openapi3.NewInt64Schema()).
		WithProperty("limit", openapi3.NewInt64Schema()).
		WithProperty("total", openapi3.NewInt64Schema()).
		WithProperty("totalPages", openapi3.NewInt64Schema())
}

// resolveSchema defines segment-resolve response schema values.
func resolveSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("segmentId", openapi3.NewStringSchema()).
		WithProperty("contactIds", openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema()))
}

// countSchema defines segment-count response schema values.
func countSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("segmentId", openapi3.NewStringSchema()).
		WithProperty("count", openapi3.NewInt64Schema())
}

// previewCountSchema defines segment preview-count response schema values.
func previewCountSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("count", openapi3.NewInt64Schema())
}

// previewCountOperation builds preview-count OpenAPI operations.
func previewCountOperation() *openapi3.Operation {
	operation := baseOperation("SegmentController_previewCount", "Preview segment contact count without saving")
	operation.Responses = openapi3.NewResponses(
		openapi3.WithStatus(200, jsonResponse("Preview count.", "#/components/schemas/SegmentPreviewCount")),
		openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
		openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
		openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		openapi3.WithStatus(503, responseWithDescription("Segment backend unavailable.")),
	)

	return operation
}

// deleteSchema defines delete response schema values.
func deleteSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().WithProperty("status", openapi3.NewStringSchema().WithDefault("deleted"))
}

// bearerSecurityRequirements builds bearer-auth operation security requirements.
func bearerSecurityRequirements() *openapi3.SecurityRequirements {
	return openapi3.NewSecurityRequirements().With(openapi3.NewSecurityRequirement().Authenticate(bearerSecurityScheme))
}

// jsonResponse builds a JSON response with a schema reference.
func jsonResponse(description string, schemaRef string) *openapi3.ResponseRef {
	return &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription(description).WithContent(openapi3.Content{
		"application/json": &openapi3.MediaType{Schema: &openapi3.SchemaRef{Ref: schemaRef}},
	})}
}

// responseWithDescription builds an OpenAPI response from a plain description.
func responseWithDescription(description string) *openapi3.ResponseRef {
	return &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription(description)}
}
