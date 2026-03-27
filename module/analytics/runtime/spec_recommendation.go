package runtime

import "github.com/getkin/kin-openapi/openapi3"

// recommendedProductSchema defines the response schema for a single recommended product.
func recommendedProductSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("id", openapi3.NewStringSchema()).
		WithProperty("name", openapi3.NewStringSchema()).
		WithProperty("price", openapi3.NewFloat64Schema()).
		WithProperty("imageUrl", openapi3.NewStringSchema()).
		WithProperty("url", openapi3.NewStringSchema())
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
				WithDescription("Single base tag shorthand (backward compatible). Merged into baseTags. Required unless baseTags or pinnedIds is set.").
				WithSchema(openapi3.NewStringSchema())},
			{Value: openapi3.NewQueryParameter("baseTags").
				WithDescription("Comma-separated base tags. Use baseTagMode to control union vs intersection matching. Required unless baseTag or pinnedIds is set.").
				WithSchema(openapi3.NewStringSchema())},
			{Value: openapi3.NewQueryParameter("baseTagMode").
				WithDescription("Controls how baseTags are matched: \"any\" (default) — products with at least one tag (union); \"all\" — products that carry every tag (intersection).").
				WithSchema(openapi3.NewStringSchema())},
			{Value: openapi3.NewQueryParameter("categoryId").
				WithDescription("Restrict candidates to one product category reference (id, slug, or name; case-insensitive for name). When the resolved category has includeChildren enabled, descendant categories are included.").
				WithSchema(openapi3.NewStringSchema())},
			{Value: openapi3.NewQueryParameter("categoryIds").
				WithDescription("Comma-separated include-category references (id, slug, or name).").
				WithSchema(openapi3.NewStringSchema())},
			{Value: openapi3.NewQueryParameter("excludeCategoryIds").
				WithDescription("Comma-separated exclude-category references (id, slug, or name). Products in these categories are removed.").
				WithSchema(openapi3.NewStringSchema())},
			{Value: openapi3.NewQueryParameter("includeTags").
				WithDescription("Comma-separated include-tag filter values. Product must contain at least one include tag.").
				WithSchema(openapi3.NewStringSchema())},
			{Value: openapi3.NewQueryParameter("excludeTags").
				WithDescription("Comma-separated exclude-tag filter values. Products containing any excluded tag are removed.").
				WithSchema(openapi3.NewStringSchema())},
			{Value: openapi3.NewQueryParameter("minPrice").
				WithDescription("Optional minimum product price filter.").
				WithSchema(openapi3.NewFloat64Schema())},
			{Value: openapi3.NewQueryParameter("maxPrice").
				WithDescription("Optional maximum product price filter.").
				WithSchema(openapi3.NewFloat64Schema())},
			{Value: openapi3.NewQueryParameter("excludePurchased").
				WithDescription("Set to \"true\" to exclude products already purchased by the contact.").
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
			{Value: openapi3.NewQueryParameter("pinnedIds").
				WithDescription("Comma-separated pinned product tokens. Supports plain <product_id> and scoped <product_id>|<variation_id> to force variation-specific URL/image resolution. baseTag/baseTags are optional when this is set.").
				WithSchema(openapi3.NewStringSchema())},
			{Value: openapi3.NewQueryParameter("excludeIds").
				WithDescription("Comma-separated exclusion tokens. Plain <product_id> removes the full product; scoped <product_id>|<variation_id> excludes only that variation from URL/image variation candidates.").
				WithSchema(openapi3.NewStringSchema())},
			{Value: openapi3.NewQueryParameter("filterVariationIds").
				WithDescription("Comma-separated variation IDs; only products linked to at least one of these variations are returned.").
				WithSchema(openapi3.NewStringSchema())},
			{Value: openapi3.NewQueryParameter("preferVariationIds").
				WithDescription("Comma-separated variation IDs; prefer the gallery image linked to a matching variation over the default first realm image.").
				WithSchema(openapi3.NewStringSchema())},
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Ranked product recommendations.", "#/components/schemas/RecommendedProduct")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request - baseTag, baseTags, or pinnedIds is required.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		),
	}
}
