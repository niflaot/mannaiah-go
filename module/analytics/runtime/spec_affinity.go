package runtime

import "github.com/getkin/kin-openapi/openapi3"

// tagAffinitySchema defines the response schema for tag affinity entries.
func tagAffinitySchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("contactId", openapi3.NewStringSchema()).
		WithProperty("tag", openapi3.NewStringSchema()).
		WithProperty("affinityScore", openapi3.NewFloat64Schema()).
		WithProperty("totalSpent", openapi3.NewFloat64Schema()).
		WithProperty("purchaseCount", openapi3.NewInt32Schema())
}

// categoryAffinitySchema defines the response schema for category affinity entries.
func categoryAffinitySchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("contactId", openapi3.NewStringSchema()).
		WithProperty("categoryId", openapi3.NewStringSchema()).
		WithProperty("categoryName", openapi3.NewStringSchema()).
		WithProperty("affinityScore", openapi3.NewFloat64Schema()).
		WithProperty("totalSpent", openapi3.NewFloat64Schema()).
		WithProperty("purchaseCount", openapi3.NewInt32Schema())
}

// variationAffinitySchema defines the response schema for variation affinity entries.
func variationAffinitySchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("contactId", openapi3.NewStringSchema()).
		WithProperty("variationName", openapi3.NewStringSchema()).
		WithProperty("variationValue", openapi3.NewStringSchema()).
		WithProperty("affinityScore", openapi3.NewFloat64Schema()).
		WithProperty("totalSpent", openapi3.NewFloat64Schema()).
		WithProperty("purchaseCount", openapi3.NewInt32Schema())
}

// affinityProfileSchema defines the response schema for a full contact affinity profile.
func affinityProfileSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("contactId", openapi3.NewStringSchema()).
		WithProperty("tags", openapi3.NewArraySchema().WithItems(tagAffinitySchema())).
		WithProperty("categories", openapi3.NewArraySchema().WithItems(categoryAffinitySchema())).
		WithProperty("variations", openapi3.NewArraySchema().WithItems(variationAffinitySchema()))
}

// affinityContactPathItem returns the path item for the full affinity profile endpoint.
func affinityContactPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{Get: getAffinityProfileOperation()}
}

// affinityTagsPathItem returns the path item for tag affinity endpoints.
func affinityTagsPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{Get: getTagAffinityOperation()}
}

// affinityCategoriesPathItem returns the path item for category affinity endpoints.
func affinityCategoriesPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{Get: getCategoryAffinityOperation()}
}

// affinityVariationsPathItem returns the path item for variation affinity endpoints.
func affinityVariationsPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{Get: getVariationAffinityOperation()}
}

// affinityRefreshPathItem returns the path item for affinity refresh endpoints.
func affinityRefreshPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{Post: affinityRefreshOperation()}
}

// affinityQueryParameters returns shared query parameters for affinity endpoints.
func affinityQueryParameters() openapi3.Parameters {
	return openapi3.Parameters{
		{Value: openapi3.NewQueryParameter("limit").WithDescription("Maximum results to return (default 10).").WithSchema(openapi3.NewInt32Schema())},
		{Value: openapi3.NewQueryParameter("minScore").WithDescription("Minimum affinity score threshold (default 0).").WithSchema(openapi3.NewFloat64Schema())},
	}
}

// getAffinityProfileOperation defines the OpenAPI operation for GET /analytics/affinity/contacts/:contactId.
func getAffinityProfileOperation() *openapi3.Operation {
	params := openapi3.Parameters{
		rfmPathParameter("contactId", "Contact ID"),
	}
	params = append(params, affinityQueryParameters()...)

	return &openapi3.Operation{
		OperationID: "AnalyticsAffinityController_getProfile",
		Summary:     "Get full affinity profile for one contact",
		Tags:        []string{analyticsTag},
		Security:    bearerSecurityRequirements(),
		Parameters:  params,
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Affinity profile.", "#/components/schemas/AffinityProfile")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		),
	}
}

// getTagAffinityOperation defines the OpenAPI operation for GET /analytics/affinity/contacts/:contactId/tags.
func getTagAffinityOperation() *openapi3.Operation {
	params := openapi3.Parameters{
		rfmPathParameter("contactId", "Contact ID"),
	}
	params = append(params, affinityQueryParameters()...)

	return &openapi3.Operation{
		OperationID: "AnalyticsAffinityController_getTagAffinity",
		Summary:     "Get tag affinity scores for one contact",
		Tags:        []string{analyticsTag},
		Security:    bearerSecurityRequirements(),
		Parameters:  params,
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Tag affinity scores.", "#/components/schemas/TagAffinity")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		),
	}
}

// getCategoryAffinityOperation defines the OpenAPI operation for GET /analytics/affinity/contacts/:contactId/categories.
func getCategoryAffinityOperation() *openapi3.Operation {
	params := openapi3.Parameters{
		rfmPathParameter("contactId", "Contact ID"),
	}
	params = append(params, affinityQueryParameters()...)

	return &openapi3.Operation{
		OperationID: "AnalyticsAffinityController_getCategoryAffinity",
		Summary:     "Get category affinity scores for one contact",
		Tags:        []string{analyticsTag},
		Security:    bearerSecurityRequirements(),
		Parameters:  params,
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Category affinity scores.", "#/components/schemas/CategoryAffinity")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		),
	}
}

// getVariationAffinityOperation defines the OpenAPI operation for GET /analytics/affinity/contacts/:contactId/variations.
func getVariationAffinityOperation() *openapi3.Operation {
	params := openapi3.Parameters{
		rfmPathParameter("contactId", "Contact ID"),
	}
	params = append(params, affinityQueryParameters()...)

	return &openapi3.Operation{
		OperationID: "AnalyticsAffinityController_getVariationAffinity",
		Summary:     "Get variation affinity scores for one contact",
		Tags:        []string{analyticsTag},
		Security:    bearerSecurityRequirements(),
		Parameters:  params,
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Variation affinity scores.", "#/components/schemas/VariationAffinity")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		),
	}
}

// affinityRefreshOperation defines the OpenAPI operation for POST /analytics/affinity/refresh.
func affinityRefreshOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "AnalyticsAffinityController_refresh",
		Summary:     "Refresh all affinity materialized views",
		Tags:        []string{analyticsTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Refresh completed.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		),
	}
}
