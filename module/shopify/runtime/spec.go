package runtime

import "github.com/getkin/kin-openapi/openapi3"

const (
	// shopifyTag defines OpenAPI tags used by Shopify endpoints.
	shopifyTag = "shopify"
	// shopifyExtensionTag defines OpenAPI tags used by Shopify Admin extension endpoints.
	shopifyExtensionTag = "shopify-extension"
	// bearerSecurityScheme defines OpenAPI security scheme keys used for bearer auth.
	bearerSecurityScheme = "shopifyBearer"
	// sessionBearerSecurityScheme defines OpenAPI security scheme keys used for Admin extension session-token auth.
	sessionBearerSecurityScheme = "shopifySessionBearer"

	shopifyManualSyncRequestSchemaRef       = "#/components/schemas/ShopifyManualSyncRequest"
	shopifyContactSyncSummarySchemaRef      = "#/components/schemas/ShopifyContactSyncSummary"
	shopifyOrderSyncSummarySchemaRef        = "#/components/schemas/ShopifyOrderSyncSummary"
	shopifyOAuthCallbackResponseSchemaRef   = "#/components/schemas/ShopifyOAuthCallbackResponse"
	shopifyWebhookPayloadSchemaRef          = "#/components/schemas/ShopifyWebhookPayload"
	shopifyExtensionOrderSummarySchemaRef   = "#/components/schemas/ShopifyExtensionOrderSummary"
	shopifyExtensionContactSummarySchemaRef = "#/components/schemas/ShopifyExtensionContactSummary"
)

// OpenAPISpec returns Shopify module OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	components := openapi3.NewComponents()
	components.SecuritySchemes = openapi3.SecuritySchemes{
		bearerSecurityScheme:        &openapi3.SecuritySchemeRef{Value: openapi3.NewJWTSecurityScheme()},
		sessionBearerSecurityScheme: &openapi3.SecuritySchemeRef{Value: openapi3.NewJWTSecurityScheme()},
	}
	components.Schemas = openapi3.Schemas{
		"ShopifyManualSyncRequest":       {Value: shopifyManualSyncRequestSchema()},
		"ShopifyContactSyncSummary":      {Value: shopifyContactSyncSummarySchema()},
		"ShopifyOrderSyncSummary":        {Value: shopifyOrderSyncSummarySchema()},
		"ShopifyOAuthCallbackResponse":   {Value: shopifyOAuthCallbackResponseSchema()},
		"ShopifyWebhookPayload":          {Value: shopifyWebhookPayloadSchema()},
		"ShopifyExtensionOrderSummary":   {Value: shopifyExtensionOrderSummarySchema()},
		"ShopifyExtensionContactSummary": {Value: shopifyExtensionContactSummarySchema()},
	}

	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info:    &openapi3.Info{Title: "Shopify API", Version: "2.2.0"},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/shopify/app", &openapi3.PathItem{Get: appLaunchOperation()}),
			openapi3.WithPath("/shopify/oauth/install", &openapi3.PathItem{Get: installOAuthOperation()}),
			openapi3.WithPath("/shopify/oauth/callback", &openapi3.PathItem{Get: oauthCallbackOperation()}),
			openapi3.WithPath("/shopify/sync/contacts", &openapi3.PathItem{Post: syncContactsOperation()}),
			openapi3.WithPath("/shopify/sync/orders", &openapi3.PathItem{Post: syncOrdersOperation()}),
			openapi3.WithPath("/shopify/webhooks", &openapi3.PathItem{Post: webhookOperation()}),
			openapi3.WithPath("/shopify/ext/orders/{shopifyOrderId}", &openapi3.PathItem{Get: extensionOrderSummaryOperation()}),
			openapi3.WithPath("/shopify/ext/orders/{shopifyOrderId}/sync", &openapi3.PathItem{Post: extensionOrderSyncOperation()}),
			openapi3.WithPath("/shopify/ext/contacts/{shopifyCustomerId}", &openapi3.PathItem{Get: extensionContactSummaryOperation()}),
			openapi3.WithPath("/shopify/ext/contacts/{shopifyCustomerId}/sync", &openapi3.PathItem{Post: extensionContactSyncOperation()}),
		),
		Components: &components,
		Tags: openapi3.Tags{
			&openapi3.Tag{Name: shopifyTag},
			&openapi3.Tag{Name: shopifyExtensionTag},
		},
	}
}

// appLaunchOperation defines OpenAPI operations for Shopify App URL launch landings.
func appLaunchOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShopifyApp_launch",
		Summary:     "Render Shopify app launch landing page",
		Description: "Returns a simple HTML page when Shopify opens the configured App URL after installation or from the Admin app launcher.",
		Tags:        []string{shopifyTag},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("HTML launch landing page returned.")),
		),
	}
}

// installOAuthOperation defines OpenAPI operations for Shopify OAuth install redirects.
func installOAuthOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShopifyOAuth_install",
		Summary:     "Start Shopify OAuth installation",
		Description: "Redirects a Shopify merchant to the app authorization flow for one shop domain.",
		Tags:        []string{shopifyTag},
		Parameters: openapi3.Parameters{
			requiredQueryParameter("shop", "Shopify store domain, for example flock-6591.myshopify.com.", openapi3.NewStringSchema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(302, responseWithDescription("Redirected to Shopify OAuth authorization.")),
			openapi3.WithStatus(400, responseWithDescription("Invalid Shopify shop domain.")),
			openapi3.WithStatus(500, responseWithDescription("Public callback URL could not be resolved.")),
			openapi3.WithStatus(503, responseWithDescription("Shopify OAuth is unavailable.")),
		),
	}
}

// oauthCallbackOperation defines OpenAPI operations for Shopify OAuth callback handling.
func oauthCallbackOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShopifyOAuth_callback",
		Summary:     "Complete Shopify OAuth installation",
		Description: "Exchanges the authorization code for one offline access token, persists the installation, and registers required webhooks.",
		Tags:        []string{shopifyTag},
		Parameters: openapi3.Parameters{
			requiredQueryParameter("code", "Shopify OAuth authorization code.", openapi3.NewStringSchema()),
			requiredQueryParameter("shop", "Shopify store domain, for example flock-6591.myshopify.com.", openapi3.NewStringSchema()),
			requiredQueryParameter("state", "State nonce generated during the install redirect.", openapi3.NewStringSchema()),
			requiredQueryParameter("hmac", "Shopify callback HMAC signature.", openapi3.NewStringSchema()),
			requiredQueryParameter("timestamp", "Unix timestamp emitted by Shopify for callback freshness validation.", openapi3.NewStringSchema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(302, responseWithDescription("Redirected to the Shopify app landing page.")),
			openapi3.WithStatus(200, jsonResponse("OAuth callback completed successfully.", shopifyOAuthCallbackResponseSchemaRef)),
			openapi3.WithStatus(400, responseWithDescription("Invalid OAuth callback request.")),
			openapi3.WithStatus(401, responseWithDescription("OAuth callback authentication failed.")),
			openapi3.WithStatus(500, responseWithDescription("Public webhook callback URL could not be resolved.")),
			openapi3.WithStatus(503, responseWithDescription("Shopify OAuth exchange or webhook registration is unavailable.")),
		),
	}
}

// syncContactsOperation defines OpenAPI operations for manual contact sync endpoints.
func syncContactsOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShopifySyncController_triggerContactSync",
		Summary:     "Trigger Shopify contact sync",
		Description: "Triggers one targeted Shopify contact synchronization by Shopify customer identifier, optionally scoped to one installed store.",
		Tags:        []string{shopifyTag},
		Security:    bearerSecurityRequirements(),
		RequestBody: jsonRequestBodyRef(shopifyManualSyncRequestSchemaRef),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Contact sync completed successfully.", shopifyContactSyncSummarySchemaRef)),
			openapi3.WithStatus(400, responseWithDescription("Invalid contact sync request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - insufficient permissions.")),
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
		Description: "Triggers one targeted Shopify order synchronization by Shopify order identifier, optionally scoped to one installed store.",
		Tags:        []string{shopifyTag},
		Security:    bearerSecurityRequirements(),
		RequestBody: jsonRequestBodyRef(shopifyManualSyncRequestSchemaRef),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Order sync completed successfully.", shopifyOrderSyncSummarySchemaRef)),
			openapi3.WithStatus(400, responseWithDescription("Invalid order sync request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - insufficient permissions.")),
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
		Description: "Receives authenticated Shopify webhooks, deduplicates deliveries, updates uninstall state, and enqueues order or customer sync jobs.",
		Tags:        []string{shopifyTag},
		Parameters: openapi3.Parameters{
			requiredHeaderParameter("X-Shopify-Hmac-Sha256", "Shopify webhook HMAC signature.", openapi3.NewStringSchema()),
			requiredHeaderParameter("X-Shopify-Webhook-Id", "Unique Shopify webhook delivery identifier.", openapi3.NewStringSchema()),
			requiredHeaderParameter("X-Shopify-Topic", "Shopify webhook topic.", openapi3.NewStringSchema()),
			requiredHeaderParameter("X-Shopify-Shop-Domain", "Shopify store domain that emitted the webhook.", openapi3.NewStringSchema()),
		},
		RequestBody: jsonRequestBodyRef(shopifyWebhookPayloadSchemaRef),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Webhook accepted.")),
			openapi3.WithStatus(400, responseWithDescription("Invalid webhook payload or headers.")),
			openapi3.WithStatus(401, responseWithDescription("Invalid webhook signature.")),
			openapi3.WithStatus(503, responseWithDescription("Webhook processor or Shopify integration is unavailable.")),
		),
	}
}

// extensionOrderSummaryOperation defines OpenAPI operations for order summaries inside the Shopify Admin extension.
func extensionOrderSummaryOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShopifyExtension_getOrderSummary",
		Summary:     "Get linked Mannaiah order summary",
		Description: "Returns the Mannaiah order linked to one Shopify order for the authenticated Shopify Admin extension session.",
		Tags:        []string{shopifyExtensionTag},
		Security:    sessionBearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathParameter("shopifyOrderId", "Shopify order identifier.", openapi3.NewStringSchema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Extension order summary resolved successfully.", shopifyExtensionOrderSummarySchemaRef)),
			openapi3.WithStatus(401, responseWithDescription("Invalid or missing Shopify Admin session token.")),
			openapi3.WithStatus(503, responseWithDescription("Shopify installation resolution or Mannaiah integration is unavailable.")),
		),
	}
}

// extensionOrderSyncOperation defines OpenAPI operations for order sync actions inside the Shopify Admin extension.
func extensionOrderSyncOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShopifyExtension_syncOrder",
		Summary:     "Sync one Shopify order from the Admin extension",
		Description: "Triggers one targeted order synchronization for the authenticated Shopify Admin extension store session.",
		Tags:        []string{shopifyExtensionTag},
		Security:    sessionBearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathParameter("shopifyOrderId", "Shopify order identifier.", openapi3.NewStringSchema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Order sync completed successfully.", shopifyOrderSyncSummarySchemaRef)),
			openapi3.WithStatus(400, responseWithDescription("Invalid Shopify order identifier.")),
			openapi3.WithStatus(401, responseWithDescription("Invalid or missing Shopify Admin session token.")),
			openapi3.WithStatus(404, responseWithDescription("Shopify order not found.")),
			openapi3.WithStatus(503, responseWithDescription("Shopify integration unavailable or disabled.")),
		),
	}
}

// extensionContactSummaryOperation defines OpenAPI operations for contact summaries inside the Shopify Admin extension.
func extensionContactSummaryOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShopifyExtension_getContactSummary",
		Summary:     "Get linked Mannaiah contact summary",
		Description: "Returns the Mannaiah contact linked to one Shopify customer for the authenticated Shopify Admin extension session.",
		Tags:        []string{shopifyExtensionTag},
		Security:    sessionBearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathParameter("shopifyCustomerId", "Shopify customer identifier.", openapi3.NewStringSchema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Extension contact summary resolved successfully.", shopifyExtensionContactSummarySchemaRef)),
			openapi3.WithStatus(401, responseWithDescription("Invalid or missing Shopify Admin session token.")),
			openapi3.WithStatus(503, responseWithDescription("Shopify installation resolution or Mannaiah integration is unavailable.")),
		),
	}
}

// extensionContactSyncOperation defines OpenAPI operations for contact sync actions inside the Shopify Admin extension.
func extensionContactSyncOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShopifyExtension_syncContact",
		Summary:     "Sync one Shopify customer from the Admin extension",
		Description: "Triggers one targeted contact synchronization for the authenticated Shopify Admin extension store session.",
		Tags:        []string{shopifyExtensionTag},
		Security:    sessionBearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathParameter("shopifyCustomerId", "Shopify customer identifier.", openapi3.NewStringSchema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Contact sync completed successfully.", shopifyContactSyncSummarySchemaRef)),
			openapi3.WithStatus(400, responseWithDescription("Invalid Shopify customer identifier.")),
			openapi3.WithStatus(401, responseWithDescription("Invalid or missing Shopify Admin session token.")),
			openapi3.WithStatus(404, responseWithDescription("Shopify customer not found.")),
			openapi3.WithStatus(503, responseWithDescription("Shopify integration unavailable or disabled.")),
		),
	}
}

// bearerSecurityRequirements builds bearer-auth operation security requirements.
func bearerSecurityRequirements() *openapi3.SecurityRequirements {
	return openapi3.NewSecurityRequirements().With(openapi3.NewSecurityRequirement().Authenticate(bearerSecurityScheme))
}

// sessionBearerSecurityRequirements builds session bearer-auth operation security requirements.
func sessionBearerSecurityRequirements() *openapi3.SecurityRequirements {
	return openapi3.NewSecurityRequirements().With(openapi3.NewSecurityRequirement().Authenticate(sessionBearerSecurityScheme))
}

// responseWithDescription builds OpenAPI responses from plain descriptions.
func responseWithDescription(description string) *openapi3.ResponseRef {
	return &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription(description)}
}

// jsonResponse builds a JSON response with a schema reference.
func jsonResponse(description string, schemaRef string) *openapi3.ResponseRef {
	return &openapi3.ResponseRef{
		Value: openapi3.NewResponse().
			WithDescription(description).
			WithContent(openapi3.Content{
				"application/json": &openapi3.MediaType{Schema: &openapi3.SchemaRef{Ref: schemaRef}},
			}),
	}
}

// jsonRequestBodyRef builds a required JSON request body referencing a component schema.
func jsonRequestBodyRef(schemaRef string) *openapi3.RequestBodyRef {
	return &openapi3.RequestBodyRef{
		Value: openapi3.NewRequestBody().
			WithRequired(true).
			WithContent(openapi3.Content{
				"application/json": &openapi3.MediaType{Schema: &openapi3.SchemaRef{Ref: schemaRef}},
			}),
	}
}

// queryParameter builds optional query-parameter OpenAPI definitions.
func queryParameter(name, description string, schema *openapi3.Schema) *openapi3.ParameterRef {
	return &openapi3.ParameterRef{Value: openapi3.NewQueryParameter(name).WithDescription(description).WithSchema(schema)}
}

// requiredQueryParameter builds required query-parameter OpenAPI definitions.
func requiredQueryParameter(name, description string, schema *openapi3.Schema) *openapi3.ParameterRef {
	parameter := openapi3.NewQueryParameter(name).WithDescription(description).WithSchema(schema)
	parameter.Required = true
	return &openapi3.ParameterRef{Value: parameter}
}

// pathParameter builds required path-parameter OpenAPI definitions.
func pathParameter(name, description string, schema *openapi3.Schema) *openapi3.ParameterRef {
	return &openapi3.ParameterRef{Value: openapi3.NewPathParameter(name).WithDescription(description).WithSchema(schema)}
}

// requiredHeaderParameter builds required header-parameter OpenAPI definitions.
func requiredHeaderParameter(name, description string, schema *openapi3.Schema) *openapi3.ParameterRef {
	parameter := openapi3.NewHeaderParameter(name).WithDescription(description).WithSchema(schema)
	parameter.Required = true
	return &openapi3.ParameterRef{Value: parameter}
}

func shopifyManualSyncRequestSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("id", openapi3.NewStringSchema()).
		WithProperty("shopDomain", openapi3.NewStringSchema()).
		WithRequired([]string{"id"})
}

func shopifyContactSyncSummarySchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("runId", openapi3.NewStringSchema()).
		WithProperty("trigger", openapi3.NewStringSchema()).
		WithProperty("processed", openapi3.NewInt32Schema()).
		WithProperty("succeeded", openapi3.NewInt32Schema()).
		WithProperty("failed", openapi3.NewInt32Schema()).
		WithProperty("skipped", openapi3.NewInt32Schema()).
		WithProperty("contactId", openapi3.NewStringSchema()).
		WithRequired([]string{"runId", "trigger", "processed", "succeeded", "failed", "skipped"})
}

func shopifyOrderSyncSummarySchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("runId", openapi3.NewStringSchema()).
		WithProperty("trigger", openapi3.NewStringSchema()).
		WithProperty("processed", openapi3.NewInt32Schema()).
		WithProperty("succeeded", openapi3.NewInt32Schema()).
		WithProperty("failed", openapi3.NewInt32Schema()).
		WithProperty("skipped", openapi3.NewInt32Schema()).
		WithProperty("orderId", openapi3.NewStringSchema()).
		WithProperty("contactId", openapi3.NewStringSchema()).
		WithRequired([]string{"runId", "trigger", "processed", "succeeded", "failed", "skipped"})
}

func shopifyOAuthCallbackResponseSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("shopDomain", openapi3.NewStringSchema()).
		WithProperty("scopes", openapi3.NewStringSchema()).
		WithProperty("installedAt", openapi3.NewDateTimeSchema()).
		WithProperty("webhooksRegistered", openapi3.NewBoolSchema()).
		WithRequired([]string{"shopDomain", "scopes", "installedAt", "webhooksRegistered"})
}

func shopifyWebhookPayloadSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("id", openapi3.NewInt64Schema()).
		WithAdditionalProperties(nil).
		WithRequired([]string{"id"})
}

func shopifyExtensionOrderSummarySchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("linked", openapi3.NewBoolSchema()).
		WithProperty("mannaiahId", openapi3.NewStringSchema()).
		WithProperty("status", openapi3.NewStringSchema()).
		WithProperty("contactName", openapi3.NewStringSchema()).
		WithProperty("lastSyncedAt", openapi3.NewDateTimeSchema()).
		WithRequired([]string{"linked"})
}

func shopifyExtensionContactSummarySchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("linked", openapi3.NewBoolSchema()).
		WithProperty("mannaiahId", openapi3.NewStringSchema()).
		WithProperty("displayName", openapi3.NewStringSchema()).
		WithProperty("lastSyncedAt", openapi3.NewDateTimeSchema()).
		WithRequired([]string{"linked"})
}
