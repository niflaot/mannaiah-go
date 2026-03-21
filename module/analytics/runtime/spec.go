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
	components.Schemas = openapi3.Schemas{
		"AnalyticsStatus":     {Value: analyticsStatusSchema()},
		"AnalyticsSeed":       {Value: analyticsSeedSchema()},
		"RFMBandConfig":       {Value: rfmBandConfigSchema()},
		"RFMBandUpdateRequest": {Value: rfmBandUpdateRequestSchema()},
		"RFMGroup":            {Value: rfmGroupSchema()},
		"RFMGroupRequest":     {Value: rfmGroupRequestSchema()},
		"RFMScore":            {Value: rfmScoreSchema()},
		"RFMScoreBatchRequest": {Value: rfmScoreBatchRequestSchema()},
		"TagAffinity":        {Value: tagAffinitySchema()},
		"CategoryAffinity":   {Value: categoryAffinitySchema()},
		"VariationAffinity":  {Value: variationAffinitySchema()},
		"AffinityProfile":    {Value: affinityProfileSchema()},
		"RecommendedProduct": {Value: recommendedProductSchema()},
	}

	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:   "Analytics API",
			Version: "2.6.0",
		},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/analytics/status", &openapi3.PathItem{Get: statusOperation()}),
			openapi3.WithPath("/analytics/seed", &openapi3.PathItem{Post: seedOperation()}),
			openapi3.WithPath("/analytics/rfm/bands", rfmBandsPathItem()),
			openapi3.WithPath("/analytics/rfm/bands/{dimension}", rfmBandByDimensionPathItem()),
			openapi3.WithPath("/analytics/rfm/groups", rfmGroupsPathItem()),
			openapi3.WithPath("/analytics/rfm/groups/{id}", rfmGroupByIDPathItem()),
			openapi3.WithPath("/analytics/rfm/contacts/{contactId}/score", rfmContactScorePathItem()),
			openapi3.WithPath("/analytics/rfm/contacts/score-batch", rfmScoreBatchPathItem()),
			openapi3.WithPath("/analytics/rfm/refresh", rfmRefreshPathItem()),
			openapi3.WithPath("/analytics/affinity/contacts/{contactId}", affinityContactPathItem()),
			openapi3.WithPath("/analytics/affinity/contacts/{contactId}/tags", affinityTagsPathItem()),
			openapi3.WithPath("/analytics/affinity/contacts/{contactId}/categories", affinityCategoriesPathItem()),
			openapi3.WithPath("/analytics/affinity/contacts/{contactId}/variations", affinityVariationsPathItem()),
			openapi3.WithPath("/analytics/affinity/refresh", affinityRefreshPathItem()),
			openapi3.WithPath("/analytics/recommendations/contacts/{contactId}", recommendationContactPathItem()),
		),
		Components: &components,
		Tags: openapi3.Tags{
			{Name: analyticsTag},
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
			openapi3.WithStatus(200, jsonResponse("Analytics status.", "#/components/schemas/AnalyticsStatus")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
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
			openapi3.WithStatus(200, jsonResponse("Analytics seed summary.", "#/components/schemas/AnalyticsSeed")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(503, responseWithDescription("Analytics backend unavailable or disabled.")),
		),
	}
}

// analyticsStatusSchema defines analytics status response schema values.
func analyticsStatusSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("enabled", openapi3.NewBoolSchema()).
		WithProperty("backendHealthy", openapi3.NewBoolSchema()).
		WithProperty("error", openapi3.NewStringSchema())
}

// analyticsSeedSchema defines analytics seed summary schema values.
func analyticsSeedSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("contacts", openapi3.NewInt64Schema()).
		WithProperty("orders", openapi3.NewInt64Schema()).
		WithProperty("orderItems", openapi3.NewInt64Schema()).
		WithProperty("membershipEvents", openapi3.NewInt64Schema()).
		WithProperty("campaignEvents", openapi3.NewInt64Schema())
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
