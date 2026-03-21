package runtime

import "github.com/getkin/kin-openapi/openapi3"

// recommendedProductSchema defines the response schema for a single recommended product.
func recommendedProductSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("id", openapi3.NewStringSchema()).
		WithProperty("name", openapi3.NewStringSchema()).
		WithProperty("price", openapi3.NewFloat64Schema()).
		WithProperty("imageUrl", openapi3.NewStringSchema())
}

// recommendationContactPathItem returns the path item for the contact recommendation endpoint.
func recommendationContactPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{Get: getRecommendationsOperation()}
}

// getRecommendationsOperation defines the OpenAPI operation for GET /analytics/recommendations/contacts/:contactId.
func getRecommendationsOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "AnalyticsRecommendationController_getRecommendations",
		Summary:     "Get ranked product recommendations for one contact",
		Tags:        []string{analyticsTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			rfmPathParameter("contactId", "Contact ID"),
			{Value: openapi3.NewQueryParameter("baseTag").
				WithRequired(true).
				WithDescription("Required product base tag; only products with this tag are candidates.").
				WithSchema(openapi3.NewStringSchema())},
			{Value: openapi3.NewQueryParameter("categoryId").
				WithDescription("Restrict candidates to one product category identifier.").
				WithSchema(openapi3.NewStringSchema())},
			{Value: openapi3.NewQueryParameter("realm").
				WithDescription("Display realm for name and image resolution (default: \"default\").").
				WithSchema(openapi3.NewStringSchema())},
			{Value: openapi3.NewQueryParameter("limit").
				WithDescription("Maximum number of products to return [1, 10] (default: 3).").
				WithSchema(openapi3.NewInt32Schema())},
			{Value: openapi3.NewQueryParameter("affinity").
				WithDescription("Set to \"true\" to enable contact-affinity-driven filtering.").
				WithSchema(openapi3.NewStringSchema())},
			{Value: openapi3.NewQueryParameter("minScore").
				WithDescription("Minimum affinity score percentile [0, 100] when affinity filtering is enabled (default: 0).").
				WithSchema(openapi3.NewFloat64Schema())},
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Ranked product recommendations.", "#/components/schemas/RecommendedProduct")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request - baseTag is required.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		),
	}
}
