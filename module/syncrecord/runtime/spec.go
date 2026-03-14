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

	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:   "Sync Record API",
			Version: "2.0.3",
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
			openapi3.WithStatus(200, responseWithDescription("Return sync runs.")),
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
			openapi3.WithStatus(200, responseWithDescription("Return sync run.")),
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
			openapi3.WithStatus(200, responseWithDescription("Return sync run stats.")),
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
