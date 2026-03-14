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

	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info:    &openapi3.Info{Title: "Segment API", Version: "2.0.4"},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/segments", &openapi3.PathItem{Post: operation("SegmentController_create", "Create segment"), Get: operation("SegmentController_list", "List segments")}),
			openapi3.WithPath("/segments/{id}", &openapi3.PathItem{Get: operation("SegmentController_get", "Get segment"), Patch: operation("SegmentController_update", "Update segment"), Delete: operation("SegmentController_delete", "Delete segment")}),
			openapi3.WithPath("/segments/{id}/resolve", &openapi3.PathItem{Post: operation("SegmentController_resolve", "Resolve segment")}),
			openapi3.WithPath("/segments/{id}/count", &openapi3.PathItem{Get: operation("SegmentController_count", "Count segment contacts")}),
		),
		Components: &components,
		Tags:       openapi3.Tags{&openapi3.Tag{Name: segmentTag}},
	}
}

// operation builds one standard segment operation.
func operation(id string, summary string) *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: id,
		Summary:     summary,
		Tags:        []string{segmentTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Success.")),
		),
	}
}

// bearerSecurityRequirements builds bearer-auth operation security requirements.
func bearerSecurityRequirements() *openapi3.SecurityRequirements {
	return openapi3.NewSecurityRequirements().With(openapi3.NewSecurityRequirement().Authenticate(bearerSecurityScheme))
}

// responseWithDescription builds an OpenAPI response from a plain description.
func responseWithDescription(description string) *openapi3.ResponseRef {
	return &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription(description)}
}
