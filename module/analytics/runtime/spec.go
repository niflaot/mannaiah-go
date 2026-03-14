package runtime

import "github.com/getkin/kin-openapi/openapi3"

const (
	// analyticsTag defines analytics OpenAPI tag values.
	analyticsTag = "analytics"
	// bearerSecurityScheme defines security-scheme key values.
	bearerSecurityScheme = "analytics_bearer"
)

// OpenAPISpec returns analytics-module OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	components := openapi3.NewComponents()
	components.SecuritySchemes = openapi3.SecuritySchemes{
		bearerSecurityScheme: &openapi3.SecuritySchemeRef{Value: openapi3.NewJWTSecurityScheme()},
	}

	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:   "Analytics API",
			Version: "2.0.4",
		},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/analytics/status", &openapi3.PathItem{Get: statusOperation()}),
			openapi3.WithPath("/analytics/seed", &openapi3.PathItem{Post: seedOperation()}),
		),
		Components: &components,
		Tags: openapi3.Tags{
			&openapi3.Tag{Name: analyticsTag},
		},
	}
}

// statusOperation defines the OpenAPI operation for status requests.
func statusOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "AnalyticsController_status",
		Summary:     "Get analytics module status",
		Tags:        []string{analyticsTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Analytics status.")),
		),
	}
}

// seedOperation defines the OpenAPI operation for seed requests.
func seedOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "AnalyticsController_seed",
		Summary:     "Seed analytics model",
		Tags:        []string{analyticsTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Analytics seed summary.")),
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
