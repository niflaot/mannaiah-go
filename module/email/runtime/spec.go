package runtime

import "github.com/getkin/kin-openapi/openapi3"

const (
	// emailTag defines email OpenAPI tag values.
	emailTag = "email"
	// bearerSecurityScheme defines security-scheme key values.
	bearerSecurityScheme = "email_bearer"
)

// OpenAPISpec returns email-module OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	components := openapi3.NewComponents()
	components.SecuritySchemes = openapi3.SecuritySchemes{
		bearerSecurityScheme: &openapi3.SecuritySchemeRef{Value: openapi3.NewJWTSecurityScheme()},
	}

	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:   "Email API",
			Version: "2.0.0",
		},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/email/send", &openapi3.PathItem{Post: sendOperation()}),
			openapi3.WithPath("/email/deliveries/{id}", &openapi3.PathItem{Get: deliveryOperation()}),
			openapi3.WithPath("/email/webhooks/ses", &openapi3.PathItem{Post: webhookOperation()}),
		),
		Components: &components,
		Tags: openapi3.Tags{
			&openapi3.Tag{Name: emailTag},
		},
	}
}

// sendOperation defines the OpenAPI operation for send requests.
func sendOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "EmailController_send",
		Summary:     "Send one email",
		Tags:        []string{emailTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(202, responseWithDescription("Email accepted.")),
		),
	}
}

// deliveryOperation defines the OpenAPI operation for delivery lookup requests.
func deliveryOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "EmailController_delivery",
		Summary:     "Get one email delivery",
		Tags:        []string{emailTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Email delivery.")),
		),
	}
}

// webhookOperation defines the OpenAPI operation for SES webhook requests.
func webhookOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "EmailController_webhook",
		Summary:     "Receive SES webhook notifications",
		Tags:        []string{emailTag},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Webhook accepted.")),
		),
	}
}

// bearerSecurityRequirements builds bearer-auth operation security requirements.
func bearerSecurityRequirements() *openapi3.SecurityRequirements {
	return openapi3.NewSecurityRequirements().With(openapi3.NewSecurityRequirement().Authenticate(bearerSecurityScheme))
}

// responseWithDescription builds an OpenAPI response from a plain description.
func responseWithDescription(description string) *openapi3.ResponseRef {
	return &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription(description)}
}
