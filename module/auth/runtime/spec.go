package runtime

import "github.com/getkin/kin-openapi/openapi3"

const (
	// statusTag defines the OpenAPI tag used by auth check endpoints.
	statusTag = "Status"
	// bearerSecurityScheme defines the OpenAPI security scheme key used for bearer auth.
	bearerSecurityScheme = "auth_bearer"
)

// OpenAPISpec returns auth-module OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	components := openapi3.NewComponents()
	components.SecuritySchemes = openapi3.SecuritySchemes{
		bearerSecurityScheme: &openapi3.SecuritySchemeRef{Value: openapi3.NewJWTSecurityScheme()},
	}

	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:   "Auth API",
			Version: "0.0.1",
		},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/check-auth", &openapi3.PathItem{Get: checkAuthOperation()}),
		),
		Components: &components,
		Tags: openapi3.Tags{
			&openapi3.Tag{Name: statusTag},
		},
	}
}

// checkAuthOperation defines the OpenAPI operation for authentication status checks.
func checkAuthOperation() *openapi3.Operation {
	statusSchema := openapi3.NewStringSchema()
	statusSchema.Example = "authenticated"

	return &openapi3.Operation{
		OperationID: "StatusController_checkAuth",
		Summary:     "Check authentication status",
		Tags:        []string{statusTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, &openapi3.ResponseRef{
				Value: openapi3.NewResponse().
					WithDescription("The user is authenticated.").
					WithContent(openapi3.Content{
						"application/json": &openapi3.MediaType{Schema: &openapi3.SchemaRef{Value: openapi3.NewObjectSchema().WithProperty("status", statusSchema)}},
					}),
			}),
			openapi3.WithStatus(401, &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription("Unauthorized. Token is missing or invalid.")}),
		),
	}
}

// bearerSecurityRequirements builds bearer-auth operation security requirements.
func bearerSecurityRequirements() *openapi3.SecurityRequirements {
	return openapi3.NewSecurityRequirements().With(openapi3.NewSecurityRequirement().Authenticate(bearerSecurityScheme))
}
