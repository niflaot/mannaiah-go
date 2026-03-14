package runtime

import "github.com/getkin/kin-openapi/openapi3"

const (
	// syncRecordTag defines sync-record OpenAPI tag values.
	syncRecordTag = "syncrecord"
	// bearerSecurityScheme defines security-scheme key values.
	bearerSecurityScheme = "syncrecord_bearer"
)

// OpenAPISpec returns sync record OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	components := openapi3.NewComponents()
	components.SecuritySchemes = openapi3.SecuritySchemes{
		bearerSecurityScheme: &openapi3.SecuritySchemeRef{Value: openapi3.NewJWTSecurityScheme()},
	}
	components.Schemas = openapi3.Schemas{
		"SyncRun":         {Value: syncRunSchema()},
		"SyncRunError":    {Value: syncRunErrorSchema()},
		"SyncRunList":     {Value: syncRunListSchema()},
		"SyncRunStats":    {Value: syncRunStatsSchema()},
		"SyncRunMetadata": {Value: openapi3.NewObjectSchema()},
	}

	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:   "Sync Record API",
			Version: "2.0.5",
		},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/syncrecord/runs", &openapi3.PathItem{Get: listRunsOperation()}),
			openapi3.WithPath("/syncrecord/runs/{id}", &openapi3.PathItem{Get: getRunOperation()}),
			openapi3.WithPath("/syncrecord/stats", &openapi3.PathItem{Get: statsOperation()}),
		),
		Components: &components,
		Tags: openapi3.Tags{
			&openapi3.Tag{Name: syncRecordTag},
		},
	}
}

// listRunsOperation defines the OpenAPI operation for listing sync runs.
func listRunsOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "SyncRecordController_findAll",
		Summary:     "List sync runs",
		Tags:        []string{syncRecordTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Return sync runs.", "#/components/schemas/SyncRunList")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden.")),
		),
	}
}

// getRunOperation defines the OpenAPI operation for retrieving one sync run.
func getRunOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "SyncRecordController_findOne",
		Summary:     "Get sync run by id",
		Tags:        []string{syncRecordTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Return sync run.", "#/components/schemas/SyncRun")),
			openapi3.WithStatus(404, responseWithDescription("Sync run not found.")),
		),
	}
}

// statsOperation defines the OpenAPI operation for sync-run stats.
func statsOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "SyncRecordController_stats",
		Summary:     "Get sync run stats",
		Tags:        []string{syncRecordTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Return sync run stats.", "#/components/schemas/SyncRunStats")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden.")),
		),
	}
}

// syncRunSchema defines sync-run response schema values.
func syncRunSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("id", openapi3.NewStringSchema()).
		WithProperty("kind", openapi3.NewStringSchema()).
		WithProperty("trigger", openapi3.NewStringSchema()).
		WithProperty("status", openapi3.NewStringSchema()).
		WithProperty("startedAt", openapi3.NewDateTimeSchema()).
		WithProperty("endedAt", openapi3.NewDateTimeSchema()).
		WithProperty("durationMs", openapi3.NewInt64Schema()).
		WithProperty("processed", openapi3.NewInt64Schema()).
		WithProperty("succeeded", openapi3.NewInt64Schema()).
		WithProperty("failed", openapi3.NewInt64Schema()).
		WithProperty("skipped", openapi3.NewInt64Schema()).
		WithProperty("errorCount", openapi3.NewInt64Schema()).
		WithProperty("metadata", openapi3.NewObjectSchema()).
		WithProperty("errors", openapi3.NewArraySchema().WithItems(syncRunErrorSchema())).
		WithProperty("createdAt", openapi3.NewDateTimeSchema()).
		WithProperty("updatedAt", openapi3.NewDateTimeSchema())
}

// syncRunErrorSchema defines sync-run error schema values.
func syncRunErrorSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("id", openapi3.NewStringSchema()).
		WithProperty("runId", openapi3.NewStringSchema()).
		WithProperty("errorType", openapi3.NewStringSchema()).
		WithProperty("errorCode", openapi3.NewStringSchema()).
		WithProperty("message", openapi3.NewStringSchema()).
		WithProperty("createdAt", openapi3.NewDateTimeSchema())
}

// syncRunListSchema defines list response schema values.
func syncRunListSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("Data", openapi3.NewArraySchema().WithItems(syncRunSchema())).
		WithProperty("Page", openapi3.NewInt64Schema()).
		WithProperty("Limit", openapi3.NewInt64Schema()).
		WithProperty("Total", openapi3.NewInt64Schema()).
		WithProperty("TotalPages", openapi3.NewInt64Schema())
}

// syncRunStatsSchema defines stats response schema values.
func syncRunStatsSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("windowStart", openapi3.NewDateTimeSchema()).
		WithProperty("totalRuns", openapi3.NewInt64Schema()).
		WithProperty("completedRuns", openapi3.NewInt64Schema()).
		WithProperty("failedRuns", openapi3.NewInt64Schema()).
		WithProperty("avgDurationMs", openapi3.NewInt64Schema()).
		WithProperty("lastFailureAt", openapi3.NewDateTimeSchema())
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
	return &openapi3.ResponseRef{
		Value: openapi3.NewResponse().WithDescription(description),
	}
}
