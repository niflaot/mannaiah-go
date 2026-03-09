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
		"FalabellaSyncStatusExecution": &openapi3.SchemaRef{Value: syncStatusExecutionSchema()},
		"FalabellaSyncStatusEntry":     &openapi3.SchemaRef{Value: syncStatusEntrySchema()},
		"FalabellaResolveResult":       &openapi3.SchemaRef{Value: resolveResultSchema()},
	}

	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:   "Falabella API",
			Version: "1.0.0",
		},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/falabella/images/transcoded", &openapi3.PathItem{Get: transcodeImageOperation()}),
			openapi3.WithPath("/falabella/brands", &openapi3.PathItem{Get: getBrandsOperation()}),
			openapi3.WithPath("/falabella/sync/products", &openapi3.PathItem{Post: syncProductsOperation()}),
			openapi3.WithPath("/falabella/sync/products/{id}", &openapi3.PathItem{Post: syncProductByIDOperation()}),
			openapi3.WithPath("/falabella/sync/status/feed/{feedId}", &openapi3.PathItem{
				Get: getSyncStatusByFeedOperation(),
			}),
			openapi3.WithPath("/falabella/sync/status/execution/{executionId}", &openapi3.PathItem{
				Get: getSyncStatusExecutionOperation(),
			}),
			openapi3.WithPath("/falabella/sync/status/execution/{executionId}/feeds", &openapi3.PathItem{
				Get: getSyncStatusByExecutionOperation(),
			}),
			openapi3.WithPath("/falabella/sync/status/product/{productId}", &openapi3.PathItem{
				Get: getSyncStatusByProductOperation(),
			}),
			openapi3.WithPath("/falabella/sync/status/feed/{feedId}/resolve", &openapi3.PathItem{
				Post: resolveFeedStatusOperation(),
			}),
		),
		Components: &components,
		Tags: openapi3.Tags{
			&openapi3.Tag{Name: falabellaTag},
		},
	}
}

// transcodeImageOperation defines OpenAPI operations for Falabella image transcode endpoints.
func transcodeImageOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "FalabellaController_transcodeImage",
		Summary:     "Transcode a source image URL to image/jpeg",
		Tags:        []string{falabellaTag},
		Parameters: openapi3.Parameters{
			&openapi3.ParameterRef{
				Value: openapi3.NewQueryParameter("src").WithDescription("Source image URL to transcode."),
			},
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("JPEG image payload.")),
			openapi3.WithStatus(400, responseWithDescription("Invalid source image URL.")),
			openapi3.WithStatus(403, responseWithDescription("Source image URL is not allowed.")),
			openapi3.WithStatus(422, responseWithDescription("Source image payload is not a decodable image.")),
			openapi3.WithStatus(503, responseWithDescription("Image transcode is disabled.")),
		),
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
		WithProperty("feedId", openapi3.NewStringSchema()).
		WithProperty("feeds", openapi3.NewArraySchema().WithItems(syncFeedSchema())).
		WithProperty("warnings", openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema())).
		WithRequired([]string{"productId", "sku", "status"})
}

// syncFeedSchema defines OpenAPI schema values for per-feed sync result values.
func syncFeedSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("step", openapi3.NewStringSchema()).
		WithProperty("task", openapi3.NewStringSchema()).
		WithProperty("action", openapi3.NewStringSchema()).
		WithProperty("feedId", openapi3.NewStringSchema()).
		WithRequired([]string{"step", "task", "feedId"})
}

// getSyncStatusByFeedOperation defines OpenAPI operations for feed sync status lookup endpoints.
func getSyncStatusByFeedOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "FalabellaController_getSyncStatusByFeed",
		Summary:     "Retrieve sync status by feed ID",
		Tags:        []string{falabellaTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			&openapi3.ParameterRef{
				Value: openapi3.NewPathParameter("feedId").WithDescription("Falabella feed ID."),
			},
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithJSONSchema("Sync status entry.", "#/components/schemas/FalabellaSyncStatusEntry")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden.")),
			openapi3.WithStatus(404, responseWithDescription("Sync entry not found.")),
		),
	}
}

// getSyncStatusByProductOperation defines OpenAPI operations for product sync status lookup endpoints.
func getSyncStatusByProductOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "FalabellaController_getSyncStatusByProduct",
		Summary:     "Retrieve sync status entries by product ID",
		Tags:        []string{falabellaTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			&openapi3.ParameterRef{
				Value: openapi3.NewPathParameter("productId").WithDescription("Product ID."),
			},
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithJSONArraySchema("Sync status entries.", "#/components/schemas/FalabellaSyncStatusEntry")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden.")),
		),
	}
}

// getSyncStatusExecutionOperation defines OpenAPI operations for execution sync status parent lookup endpoints.
func getSyncStatusExecutionOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "FalabellaController_getSyncStatusExecution",
		Summary:     "Retrieve sync execution by execution ID",
		Tags:        []string{falabellaTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			&openapi3.ParameterRef{
				Value: openapi3.NewPathParameter("executionId").WithDescription("Sync execution ID."),
			},
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithJSONSchema("Sync execution.", "#/components/schemas/FalabellaSyncStatusExecution")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden.")),
			openapi3.WithStatus(404, responseWithDescription("Sync execution not found.")),
		),
	}
}

// getSyncStatusByExecutionOperation defines OpenAPI operations for execution child feed lookup endpoints.
func getSyncStatusByExecutionOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "FalabellaController_getSyncStatusByExecution",
		Summary:     "Retrieve sync status entries by execution ID",
		Tags:        []string{falabellaTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			&openapi3.ParameterRef{
				Value: openapi3.NewPathParameter("executionId").WithDescription("Sync execution ID."),
			},
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithJSONArraySchema("Sync status entries.", "#/components/schemas/FalabellaSyncStatusEntry")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden.")),
		),
	}
}

// resolveFeedStatusOperation defines OpenAPI operations for feed status resolution endpoints.
func resolveFeedStatusOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "FalabellaController_resolveFeedStatus",
		Summary:     "Resolve Falabella feed status",
		Tags:        []string{falabellaTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			&openapi3.ParameterRef{
				Value: openapi3.NewPathParameter("feedId").WithDescription("Falabella feed ID."),
			},
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithJSONSchema("Feed resolution result.", "#/components/schemas/FalabellaResolveResult")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden.")),
			openapi3.WithStatus(404, responseWithDescription("Sync entry not found.")),
			openapi3.WithStatus(409, responseWithDescription("Feed not yet finished processing.")),
			openapi3.WithStatus(503, responseWithDescription("Falabella integration unavailable.")),
		),
	}
}

// syncStatusEntrySchema defines OpenAPI schemas for sync status entry response values.
func syncStatusEntrySchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("feedId", openapi3.NewStringSchema()).
		WithProperty("productId", openapi3.NewStringSchema()).
		WithProperty("sku", openapi3.NewStringSchema()).
		WithProperty("variationIds", openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema())).
		WithProperty("step", openapi3.NewStringSchema()).
		WithProperty("action", openapi3.NewStringSchema()).
		WithProperty("task", openapi3.NewStringSchema()).
		WithProperty("status", openapi3.NewStringSchema()).
		WithProperty("syncedAt", openapi3.NewStringSchema()).
		WithProperty("resolvedAt", openapi3.NewStringSchema()).
		WithRequired([]string{"feedId", "productId", "sku", "action", "status", "syncedAt"})
}

// syncStatusExecutionSchema defines OpenAPI schemas for sync execution response values.
func syncStatusExecutionSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("executionId", openapi3.NewStringSchema()).
		WithProperty("startedAt", openapi3.NewStringSchema()).
		WithRequired([]string{"executionId", "startedAt"})
}

// resolveResultSchema defines OpenAPI schemas for feed resolution result values.
func resolveResultSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("feedId", openapi3.NewStringSchema()).
		WithProperty("step", openapi3.NewStringSchema()).
		WithProperty("task", openapi3.NewStringSchema()).
		WithProperty("status", openapi3.NewStringSchema()).
		WithProperty("action", openapi3.NewStringSchema()).
		WithProperty("totalRecords", openapi3.NewIntegerSchema()).
		WithProperty("processedRecords", openapi3.NewIntegerSchema()).
		WithProperty("failedRecords", openapi3.NewIntegerSchema()).
		WithProperty("errors", openapi3.NewArraySchema().WithItems(feedErrorDetailSchema())).
		WithRequired([]string{"feedId", "status", "action", "totalRecords", "processedRecords", "failedRecords"})
}

// feedErrorDetailSchema defines OpenAPI schemas for feed error detail values.
func feedErrorDetailSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("code", openapi3.NewIntegerSchema()).
		WithProperty("message", openapi3.NewStringSchema()).
		WithProperty("sellerSku", openapi3.NewStringSchema()).
		WithRequired([]string{"code", "message"})
}

// responseWithJSONArraySchema builds OpenAPI JSON array responses with schema references.
func responseWithJSONArraySchema(description string, schemaRef string) *openapi3.ResponseRef {
	return &openapi3.ResponseRef{
		Value: openapi3.NewResponse().
			WithDescription(description).
			WithContent(openapi3.Content{
				"application/json": &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{
						Value: openapi3.NewArraySchema().WithItems(&openapi3.Schema{
							Extensions: map[string]any{"$ref": schemaRef},
						}),
					},
				},
			}),
	}
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
