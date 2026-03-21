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
		"Campaign":             {Value: campaignSchema()},
		"CampaignListResult":   {Value: campaignListSchema()},
		"CampaignDelete":       {Value: deleteStatusSchema()},
		"CampaignDeliveryRow":  {Value: campaignDeliveryRowSchema()},
		"CampaignDeliveryList": {Value: campaignDeliveryListSchema()},
	}

	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info:    &openapi3.Info{Title: "Campaign API", Version: "2.1.1"},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/campaigns", &openapi3.PathItem{Post: createOperation(), Get: listOperation()}),
			openapi3.WithPath("/campaigns/{id}", &openapi3.PathItem{Get: getOperation(), Patch: updateOperation(), Delete: deleteOperation()}),
			openapi3.WithPath("/campaigns/{id}/send", &openapi3.PathItem{Post: sendOperation()}),
			openapi3.WithPath("/campaigns/{id}/deliveries", &openapi3.PathItem{Get: listDeliveriesOperation()}),
		),
		Components: &components,
		Tags:       openapi3.Tags{&openapi3.Tag{Name: campaignTag}},
	}
}

// createOperation builds create campaign OpenAPI operations.
func createOperation() *openapi3.Operation {
	operation := baseOperation("CampaignController_create", "Create campaign")
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
		WithProperty("createdAt", openapi3.NewDateTimeSchema()).
		WithProperty("updatedAt", openapi3.NewDateTimeSchema())
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
