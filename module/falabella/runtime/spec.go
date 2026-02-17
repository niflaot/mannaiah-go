package runtime

import "github.com/getkin/kin-openapi/openapi3"

const (
	// falabellaTag defines OpenAPI tags used by Falabella endpoints.
	falabellaTag = "falabella"
	// bearerSecurityScheme defines OpenAPI security scheme keys used for bearer auth.
	bearerSecurityScheme = "falabellaBearer"
)

// OpenAPISpec returns Falabella module OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	components := openapi3.NewComponents()
	components.SecuritySchemes = openapi3.SecuritySchemes{
		bearerSecurityScheme: &openapi3.SecuritySchemeRef{
			Value: openapi3.NewJWTSecurityScheme(),
		},
	}
	components.Schemas = openapi3.Schemas{
		"FalabellaSyncProductsRequest": &openapi3.SchemaRef{Value: syncProductsRequestSchema()},
		"FalabellaSyncSummary":         &openapi3.SchemaRef{Value: syncSummarySchema()},
	}

	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:   "Falabella API",
			Version: "1.0.0",
		},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/falabella/brands", &openapi3.PathItem{Get: getBrandsOperation()}),
			openapi3.WithPath("/falabella/sync/products", &openapi3.PathItem{Post: syncProductsOperation()}),
			openapi3.WithPath("/falabella/sync/products/{id}", &openapi3.PathItem{Post: syncProductByIDOperation()}),
		),
		Components: &components,
		Tags: openapi3.Tags{
			&openapi3.Tag{Name: falabellaTag},
		},
	}
}

// getBrandsOperation defines OpenAPI operations for Falabella GetBrands endpoints.
func getBrandsOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "FalabellaController_getBrands",
		Summary:     "Retrieve Falabella brands",
		Tags:        []string{falabellaTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Return Falabella GetBrands payload.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(503, responseWithDescription("Falabella integration unavailable or invalid configuration.")),
		),
	}
}

// syncProductsOperation defines OpenAPI operations for Falabella batch product sync endpoints.
func syncProductsOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "FalabellaController_syncProducts",
		Summary:     "Sync products to Falabella",
		Tags:        []string{falabellaTag},
		Security:    bearerSecurityRequirements(),
		RequestBody: &openapi3.RequestBodyRef{
			Value: openapi3.NewRequestBody().
				WithDescription("Optional list of product IDs. When empty, all products are evaluated for falabella realm sync.").
				WithContent(openapi3.Content{
					"application/json": &openapi3.MediaType{
						Schema: &openapi3.SchemaRef{Ref: "#/components/schemas/FalabellaSyncProductsRequest"},
					},
				}),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithJSONSchema("Sync summary.", "#/components/schemas/FalabellaSyncSummary")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(503, responseWithDescription("Falabella integration unavailable or invalid configuration.")),
		),
	}
}

// syncProductByIDOperation defines OpenAPI operations for Falabella single-product sync endpoints.
func syncProductByIDOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "FalabellaController_syncProductByID",
		Summary:     "Sync one product to Falabella",
		Tags:        []string{falabellaTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			&openapi3.ParameterRef{
				Value: openapi3.NewPathParameter("id").
					WithDescription("Product ID."),
			},
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithJSONSchema("Sync summary.", "#/components/schemas/FalabellaSyncSummary")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(503, responseWithDescription("Falabella integration unavailable or invalid configuration.")),
		),
	}
}

// syncProductsRequestSchema defines OpenAPI schemas for batch sync request payload values.
func syncProductsRequestSchema() *openapi3.Schema {
	schema := openapi3.NewObjectSchema().
		WithProperty("ids", openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema()))
	schema.Description = "Optional list of product IDs to sync."
	return schema
}

// syncSummarySchema defines OpenAPI schemas for sync summary payload values.
func syncSummarySchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("requested", openapi3.NewIntegerSchema()).
		WithProperty("synced", openapi3.NewIntegerSchema()).
		WithProperty("skipped", openapi3.NewIntegerSchema()).
		WithProperty("failed", openapi3.NewIntegerSchema()).
		WithProperty("results", openapi3.NewArraySchema().WithItems(syncResultSchema())).
		WithRequired([]string{"requested", "synced", "skipped", "failed", "results"})
}

// syncResultSchema defines OpenAPI schema values for per-product sync results.
func syncResultSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("productId", openapi3.NewStringSchema()).
		WithProperty("sku", openapi3.NewStringSchema()).
		WithProperty("status", openapi3.NewStringSchema()).
		WithProperty("reason", openapi3.NewStringSchema()).
		WithRequired([]string{"productId", "sku", "status"})
}

// bearerSecurityRequirements builds bearer-auth operation security requirements.
func bearerSecurityRequirements() *openapi3.SecurityRequirements {
	return openapi3.NewSecurityRequirements().With(openapi3.NewSecurityRequirement().Authenticate(bearerSecurityScheme))
}

// responseWithDescription builds OpenAPI responses from plain descriptions.
func responseWithDescription(description string) *openapi3.ResponseRef {
	return &openapi3.ResponseRef{
		Value: openapi3.NewResponse().WithDescription(description),
	}
}

// responseWithJSONSchema builds OpenAPI JSON responses with schema references.
func responseWithJSONSchema(description string, schemaRef string) *openapi3.ResponseRef {
	return &openapi3.ResponseRef{
		Value: openapi3.NewResponse().
			WithDescription(description).
			WithContent(openapi3.Content{
				"application/json": &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{Ref: schemaRef},
				},
			}),
	}
}
