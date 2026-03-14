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

	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info:    &openapi3.Info{Title: "Campaign API", Version: "2.0.0"},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/campaigns", &openapi3.PathItem{Post: operation("CampaignController_create", "Create campaign"), Get: operation("CampaignController_list", "List campaigns")}),
			openapi3.WithPath("/campaigns/{id}", &openapi3.PathItem{Get: operation("CampaignController_get", "Get campaign"), Patch: operation("CampaignController_update", "Update campaign"), Delete: operation("CampaignController_delete", "Delete campaign")}),
			openapi3.WithPath("/campaigns/{id}/send", &openapi3.PathItem{Post: operation("CampaignController_send", "Send campaign")}),
		),
		Components: &components,
		Tags:       openapi3.Tags{&openapi3.Tag{Name: campaignTag}},
	}
}

// operation builds one standard campaign operation.
func operation(id string, summary string) *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: id,
		Summary:     summary,
		Tags:        []string{campaignTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Success.")),
		),
	}
}

// bearerSecurityRequirements builds bearer-auth operation security requirements.
func bearerSecurityRequirements() *openapi3.SecurityRequirements {
	return openapi3.NewSecurityRequirements().With(openapi3.NewSecurityRequirement().Authenticate(campaignBearerSecurityScheme))
}

// responseWithDescription builds an OpenAPI response from a plain description.
func responseWithDescription(description string) *openapi3.ResponseRef {
	return &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription(description)}
}
