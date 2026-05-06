package runtime

import "github.com/getkin/kin-openapi/openapi3"

const (
	// shopifyTag defines OpenAPI tags used by Shopify endpoints.
	shopifyTag = "shopify"
	// bearerSecurityScheme defines OpenAPI security scheme keys used for bearer auth.
	bearerSecurityScheme = "shopifyBearer"
)

// OpenAPISpec returns Shopify module OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	components := openapi3.NewComponents()
	components.SecuritySchemes = openapi3.SecuritySchemes{
		bearerSecurityScheme: &openapi3.SecuritySchemeRef{Value: openapi3.NewJWTSecurityScheme()},
	}

	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info:    &openapi3.Info{Title: "Shopify API", Version: "1.0.0"},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/shopify/sync/contacts", &openapi3.PathItem{Post: syncContactsOperation()}),
			openapi3.WithPath("/shopify/sync/orders", &openapi3.PathItem{Post: syncOrdersOperation()}),
			openapi3.WithPath("/shopify/webhooks", &openapi3.PathItem{Post: webhookOperation()}),
		),
		Components: &components,
		Tags:       openapi3.Tags{&openapi3.Tag{Name: shopifyTag}},
	}
}

// syncContactsOperation defines OpenAPI operations for manual contact sync endpoints.
func syncContactsOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShopifySyncController_triggerContactSync",
		Summary:     "Trigger Shopify contact sync",
		Tags:        []string{shopifyTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			queryParameter("id", "Shopify customer identifier for targeted sync.", openapi3.NewStringSchema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Sync triggered successfully.")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("Shopify customer not found.")),
			openapi3.WithStatus(503, responseWithDescription("Shopify integration unavailable or disabled.")),
		),
	}
}

// syncOrdersOperation defines OpenAPI operations for manual order sync endpoints.
func syncOrdersOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShopifySyncController_triggerOrderSync",
		Summary:     "Trigger Shopify order sync",
		Tags:        []string{shopifyTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			queryParameter("id", "Shopify order identifier for targeted sync.", openapi3.NewStringSchema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Sync triggered successfully.")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("Shopify order not found.")),
			openapi3.WithStatus(503, responseWithDescription("Shopify integration unavailable or disabled.")),
		),
	}
}

// webhookOperation defines OpenAPI operations for webhook endpoints.
func webhookOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShopifyWebhookController_receive",
		Summary:     "Receive Shopify webhooks",
		Tags:        []string{shopifyTag},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Webhook accepted.")),
			openapi3.WithStatus(401, responseWithDescription("Invalid webhook signature.")),
		),
	}
}

// bearerSecurityRequirements builds bearer-auth operation security requirements.
func bearerSecurityRequirements() *openapi3.SecurityRequirements {
	return openapi3.NewSecurityRequirements().With(openapi3.NewSecurityRequirement().Authenticate(bearerSecurityScheme))
}

// responseWithDescription builds OpenAPI responses from plain descriptions.
func responseWithDescription(description string) *openapi3.ResponseRef {
	return &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription(description)}
}

// queryParameter builds optional query-parameter OpenAPI definitions.
func queryParameter(name, description string, schema *openapi3.Schema) *openapi3.ParameterRef {
	return &openapi3.ParameterRef{Value: openapi3.NewQueryParameter(name).WithDescription(description).WithSchema(schema)}
}
