package runtime

import "github.com/getkin/kin-openapi/openapi3"

const (
	// campaignTag defines campaign OpenAPI tag values.
	campaignTag = "campaigns"
	// campaignBearerSecurityScheme defines security-scheme key values.
	campaignBearerSecurityScheme = "campaign_bearer"
)

// OpenAPISpec returns campaign-module OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	components := openapi3.NewComponents()
	components.SecuritySchemes = openapi3.SecuritySchemes{
		campaignBearerSecurityScheme: &openapi3.SecuritySchemeRef{Value: openapi3.NewJWTSecurityScheme()},
	}
	components.Schemas = openapi3.Schemas{
		"Campaign":               {Value: campaignSchema()},
		"CampaignProductBlock":   {Value: campaignProductBlockSchema()},
		"CampaignCreateRequest":  {Value: campaignCreateRequestSchema()},
		"CampaignUpdateRequest":  {Value: campaignUpdateRequestSchema()},
		"CampaignListResult":     {Value: campaignListSchema()},
		"CampaignDelete":         {Value: deleteStatusSchema()},
		"CampaignDeliveryRow":    {Value: campaignDeliveryRowSchema()},
		"CampaignDeliveryList":   {Value: campaignDeliveryListSchema()},
		"CampaignTestSendResult": {Value: campaignTestSendResultSchema()},
	}

	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info:    &openapi3.Info{Title: "Campaign API", Version: "2.5.9"},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/campaigns", &openapi3.PathItem{Post: createOperation(), Get: listOperation()}),
			openapi3.WithPath("/campaigns/{id}", &openapi3.PathItem{Get: getOperation(), Patch: updateOperation(), Delete: deleteOperation()}),
			openapi3.WithPath("/campaigns/{id}/send", &openapi3.PathItem{Post: sendOperation()}),
			openapi3.WithPath("/campaigns/{id}/test", &openapi3.PathItem{Post: testSendOperation()}),
			openapi3.WithPath("/campaigns/{id}/deliveries", &openapi3.PathItem{Get: listDeliveriesOperation()}),
		),
		Components: &components,
		Tags:       openapi3.Tags{&openapi3.Tag{Name: campaignTag}},
	}
}

// createOperation builds create campaign OpenAPI operations.
func createOperation() *openapi3.Operation {
	operation := baseOperation("CampaignController_create", "Create campaign")
	operation.RequestBody = &openapi3.RequestBodyRef{
		Value: openapi3.NewRequestBody().
			WithRequired(true).
			WithContent(openapi3.Content{
				"application/json": &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{Ref: "#/components/schemas/CampaignCreateRequest"},
				},
			}),
	}
	operation.Responses = openapi3.NewResponses(
		openapi3.WithStatus(201, jsonResponse("Campaign created.", "#/components/schemas/Campaign")),
		openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
		openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
		openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
	)

	return operation
}

// listOperation builds list campaigns OpenAPI operations.
func listOperation() *openapi3.Operation {
	operation := baseOperation("CampaignController_list", "List campaigns")
	operation.Responses = openapi3.NewResponses(
		openapi3.WithStatus(200, jsonResponse("Campaign list.", "#/components/schemas/CampaignListResult")),
		openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
		openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
	)

	return operation
}

// getOperation builds get campaign OpenAPI operations.
func getOperation() *openapi3.Operation {
	operation := baseOperation("CampaignController_get", "Get campaign")
	operation.Responses = openapi3.NewResponses(
		openapi3.WithStatus(200, jsonResponse("Campaign.", "#/components/schemas/Campaign")),
		openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
		openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		openapi3.WithStatus(404, responseWithDescription("Campaign not found.")),
	)

	return operation
}

// updateOperation builds update campaign OpenAPI operations.
func updateOperation() *openapi3.Operation {
	operation := baseOperation("CampaignController_update", "Update campaign")
	operation.RequestBody = &openapi3.RequestBodyRef{
		Value: openapi3.NewRequestBody().
			WithRequired(true).
			WithContent(openapi3.Content{
				"application/json": &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{Ref: "#/components/schemas/CampaignUpdateRequest"},
				},
			}),
	}
	operation.Responses = openapi3.NewResponses(
		openapi3.WithStatus(200, jsonResponse("Campaign updated.", "#/components/schemas/Campaign")),
		openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
		openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
		openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		openapi3.WithStatus(404, responseWithDescription("Campaign not found.")),
	)

	return operation
}

// deleteOperation builds delete campaign OpenAPI operations.
func deleteOperation() *openapi3.Operation {
	operation := baseOperation("CampaignController_delete", "Delete campaign")
	operation.Responses = openapi3.NewResponses(
		openapi3.WithStatus(200, jsonResponse("Campaign deleted.", "#/components/schemas/CampaignDelete")),
		openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
		openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		openapi3.WithStatus(404, responseWithDescription("Campaign not found.")),
	)

	return operation
}

// testSendOperation builds test-send campaign OpenAPI operations.
func testSendOperation() *openapi3.Operation {
	operation := baseOperation("CampaignController_testSend", "Test-send campaign to a single recipient")
	operation.Description = "Renders the campaign template for the given contact and delivers it to the override email address. Does not affect campaign status or counters. SES message-tag values are sanitized internally for provider compatibility. When UNSUBSCRIBE_BASE_URL and MARKETING_OPTOUT_SECRET are configured, template context injects .Custom.unsubscribe_url with a signed opt-out token."
	operation.RequestBody = &openapi3.RequestBodyRef{
		Value: openapi3.NewRequestBody().
			WithRequired(true).
			WithContent(openapi3.Content{
				"application/json": &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{
						Value: openapi3.NewObjectSchema().
							WithProperty("contactId", openapi3.NewStringSchema()).
							WithProperty("email", openapi3.NewStringSchema()),
					},
				},
			}),
	}
	operation.Responses = openapi3.NewResponses(
		openapi3.WithStatus(202, jsonResponse("Test email submitted.", "#/components/schemas/CampaignTestSendResult")),
		openapi3.WithStatus(400, responseWithDescription("Missing or invalid email address, invalid template syntax, or invalid contact personalization context.")),
		openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
		openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		openapi3.WithStatus(404, responseWithDescription("Campaign not found.")),
		openapi3.WithStatus(503, responseWithDescription("Email sender not configured or currently unavailable.")),
	)

	return operation
}

// sendOperation builds send campaign OpenAPI operations.
func sendOperation() *openapi3.Operation {
	operation := baseOperation("CampaignController_send", "Send campaign")
	operation.Responses = openapi3.NewResponses(
		openapi3.WithStatus(202, jsonResponse("Campaign accepted for delivery.", "#/components/schemas/Campaign")),
		openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
		openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		openapi3.WithStatus(404, responseWithDescription("Campaign not found.")),
		openapi3.WithStatus(409, responseWithDescription("Campaign send conflict.")),
	)

	return operation
}

// listDeliveriesOperation builds list campaign deliveries OpenAPI operations.
func listDeliveriesOperation() *openapi3.Operation {
	operation := baseOperation("CampaignController_listDeliveries", "List campaign deliveries")
	operation.Parameters = openapi3.Parameters{
		{Value: &openapi3.Parameter{Name: "page", In: "query", Schema: &openapi3.SchemaRef{Value: openapi3.NewInt64Schema()}}},
		{Value: &openapi3.Parameter{Name: "limit", In: "query", Schema: &openapi3.SchemaRef{Value: openapi3.NewInt64Schema()}}},
	}
	operation.Responses = openapi3.NewResponses(
		openapi3.WithStatus(200, jsonResponse("Campaign delivery list.", "#/components/schemas/CampaignDeliveryList")),
		openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
		openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		openapi3.WithStatus(404, responseWithDescription("Campaign not found.")),
	)

	return operation
}

// campaignDeliveryRowSchema defines a single delivery row schema.
func campaignDeliveryRowSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("contactId", openapi3.NewStringSchema()).
		WithProperty("email", openapi3.NewStringSchema()).
		WithProperty("status", openapi3.NewStringSchema()).
		WithProperty("createdAt", openapi3.NewDateTimeSchema()).
		WithProperty("updatedAt", openapi3.NewDateTimeSchema())
}

// campaignDeliveryListSchema defines delivery list response schema values.
func campaignDeliveryListSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("data", openapi3.NewArraySchema().WithItems(campaignDeliveryRowSchema())).
		WithProperty("page", openapi3.NewInt64Schema()).
		WithProperty("limit", openapi3.NewInt64Schema()).
		WithProperty("total", openapi3.NewInt64Schema()).
		WithProperty("totalPages", openapi3.NewInt64Schema())
}

// baseOperation builds one standard campaign operation.
func baseOperation(id string, summary string) *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: id,
		Summary:     summary,
		Tags:        []string{campaignTag},
		Security:    bearerSecurityRequirements(),
	}
}

// campaignSchema defines campaign response schema values.
func campaignSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("id", openapi3.NewStringSchema()).
		WithProperty("name", openapi3.NewStringSchema()).
		WithProperty("slug", openapi3.NewStringSchema()).
		WithProperty("channel", openapi3.NewStringSchema()).
		WithProperty("segmentId", openapi3.NewStringSchema()).
		WithProperty("subject", openapi3.NewStringSchema()).
		WithProperty("htmlBody", openapi3.NewStringSchema()).
		WithProperty("textBody", openapi3.NewStringSchema()).
		WithProperty("status", openapi3.NewStringSchema().WithEnum("PLANNED", "PROCESSING", "SENT", "FAILED")).
		WithProperty("totalRecipients", openapi3.NewInt64Schema()).
		WithProperty("sentCount", openapi3.NewInt64Schema()).
		WithProperty("failedCount", openapi3.NewInt64Schema()).
		WithProperty("templateVars", openapi3.NewObjectSchema().WithAdditionalProperties(openapi3.NewStringSchema())).
		WithProperty("productBlocks", openapi3.NewArraySchema().WithItems(campaignProductBlockSchema())).
		WithProperty("createdAt", openapi3.NewDateTimeSchema()).
		WithProperty("updatedAt", openapi3.NewDateTimeSchema())
}

// campaignProductBlockSchema defines campaign product-block payload schema values.
func campaignProductBlockSchema() *openapi3.Schema {
	categoryIDSchema := openapi3.NewStringSchema()
	categoryIDSchema.Description = "Optional category reference for dynamic candidates. Supports category id, slug, or name (case-insensitive). When the resolved category has includeChildren enabled, descendant categories are included."
	categoryIDsSchema := openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema())
	categoryIDsSchema.Description = "Optional include-category references. Supports category ids, slugs, or names."
	excludeCategoryIDsSchema := openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema())
	excludeCategoryIDsSchema.Description = "Optional exclude-category references. Products belonging to these categories are excluded."
	includeTagsSchema := openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema())
	includeTagsSchema.Description = "Optional include-tag filter. Product must contain at least one included tag."
	excludeTagsSchema := openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema())
	excludeTagsSchema.Description = "Optional exclude-tag filter. Products containing any excluded tag are removed."
	pinnedProductsSchema := openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema())
	pinnedProductsSchema.Description = "Pinned products. Token format supports plain <product_id> and scoped <product_id>|<variation_id>."
	excludedProductsSchema := openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema())
	excludedProductsSchema.Description = "Excluded products. Token format supports plain <product_id> and scoped <product_id>|<variation_id>."

	return openapi3.NewObjectSchema().
		WithProperty("id", openapi3.NewStringSchema()).
		WithProperty("baseTag", openapi3.NewStringSchema()).
		WithProperty("baseTags", openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema())).
		WithProperty("baseTagMode", openapi3.NewStringSchema().WithEnum("any", "all")).
		WithProperty("useAffinity", openapi3.NewBoolSchema()).
		WithProperty("affinityMinScorePct", openapi3.NewFloat64Schema()).
		WithProperty("categoryId", categoryIDSchema).
		WithProperty("categoryIds", categoryIDsSchema).
		WithProperty("excludeCategoryIds", excludeCategoryIDsSchema).
		WithProperty("includeTags", includeTagsSchema).
		WithProperty("excludeTags", excludeTagsSchema).
		WithProperty("minPrice", openapi3.NewFloat64Schema()).
		WithProperty("maxPrice", openapi3.NewFloat64Schema()).
		WithProperty("excludePurchasedProducts", openapi3.NewBoolSchema()).
		WithProperty("realm", openapi3.NewStringSchema()).
		WithProperty("limit", openapi3.NewInt64Schema()).
		WithProperty("pinnedProductIds", pinnedProductsSchema).
		WithProperty("excludeProductIds", excludedProductsSchema).
		WithProperty("filterVariationIds", openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema())).
		WithProperty("preferVariationIds", openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema()))
}

// campaignCreateRequestSchema defines create request payload schema values.
func campaignCreateRequestSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("name", openapi3.NewStringSchema()).
		WithProperty("slug", openapi3.NewStringSchema()).
		WithProperty("channel", openapi3.NewStringSchema().WithEnum("email", "all")).
		WithProperty("segmentId", openapi3.NewStringSchema()).
		WithProperty("subject", openapi3.NewStringSchema()).
		WithProperty("htmlBody", openapi3.NewStringSchema()).
		WithProperty("textBody", openapi3.NewStringSchema()).
		WithProperty("templateVars", openapi3.NewObjectSchema().WithAdditionalProperties(openapi3.NewStringSchema())).
		WithProperty("productBlocks", openapi3.NewArraySchema().WithItems(campaignProductBlockSchema()))
}

// campaignUpdateRequestSchema defines update request payload schema values.
func campaignUpdateRequestSchema() *openapi3.Schema {
	return campaignCreateRequestSchema()
}

// campaignListSchema defines campaign-list response schema values.
func campaignListSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("data", openapi3.NewArraySchema().WithItems(campaignSchema())).
		WithProperty("page", openapi3.NewInt64Schema()).
		WithProperty("limit", openapi3.NewInt64Schema()).
		WithProperty("total", openapi3.NewInt64Schema()).
		WithProperty("totalPages", openapi3.NewInt64Schema())
}

// campaignTestSendResultSchema defines test-send response schema values.
func campaignTestSendResultSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("email", openapi3.NewStringSchema()).
		WithProperty("subject", openapi3.NewStringSchema()).
		WithProperty("status", openapi3.NewStringSchema())
}

// deleteStatusSchema defines delete response schema values.
func deleteStatusSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().WithProperty("status", openapi3.NewStringSchema().WithDefault("deleted"))
}

// bearerSecurityRequirements builds bearer-auth operation security requirements.
func bearerSecurityRequirements() *openapi3.SecurityRequirements {
	return openapi3.NewSecurityRequirements().With(openapi3.NewSecurityRequirement().Authenticate(campaignBearerSecurityScheme))
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
