package runtime

import "github.com/getkin/kin-openapi/openapi3"

const (
	// wooTag defines OpenAPI tags used by WooCommerce endpoints.
	wooTag = "woocommerce"
	// bearerSecurityScheme defines OpenAPI security scheme keys used for bearer auth.
	bearerSecurityScheme = "wooBearer"
)

// OpenAPISpec returns WooCommerce module OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	components := openapi3.NewComponents()
	components.SecuritySchemes = openapi3.SecuritySchemes{
		bearerSecurityScheme: &openapi3.SecuritySchemeRef{
			Value: openapi3.NewJWTSecurityScheme(),
		},
	}

	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:   "WooCommerce API",
			Version: "1.0.0",
		},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/woo/sync/contacts", &openapi3.PathItem{
				Post: syncContactsOperation(),
			}),
			openapi3.WithPath("/woo/sync/coupons", &openapi3.PathItem{
				Post: syncCouponsOperation(),
			}),
			openapi3.WithPath("/woo/sync/orders", &openapi3.PathItem{
				Post: syncOrdersOperation(),
			}),
		),
		Components: &components,
		Tags: openapi3.Tags{
			&openapi3.Tag{Name: wooTag},
		},
	}
}

// syncCouponsOperation defines OpenAPI operations for manual coupon sync endpoints.
func syncCouponsOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "WooCommerceSyncController_triggerCouponSync",
		Summary:     "Trigger WooCommerce coupon sync",
		Tags:        []string{wooTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Sync triggered successfully.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(503, responseWithDescription("WooCommerce integration unavailable or disabled.")),
		),
	}
}

// syncOrdersOperation defines OpenAPI operations for manual order sync endpoints.
func syncOrdersOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "WooCommerceSyncController_triggerOrderSync",
		Summary:     "Trigger WooCommerce order sync",
		Tags:        []string{wooTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			queryParameter("id", "Optional WooCommerce order numeric identifier for targeted sync.", openapi3.NewIntegerSchema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Sync triggered successfully.")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("WooCommerce order not found.")),
			openapi3.WithStatus(503, responseWithDescription("WooCommerce integration unavailable or disabled.")),
		),
	}
}

// syncContactsOperation defines OpenAPI operations for manual contact sync endpoints.
func syncContactsOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "WooCommerceSyncController_triggerContactSync",
		Summary:     "Trigger WooCommerce contact sync",
		Tags:        []string{wooTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			queryParameter("email", "Optional contact email for targeted sync.", openapi3.NewStringSchema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Sync triggered successfully.")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("WooCommerce contact not found.")),
			openapi3.WithStatus(503, responseWithDescription("WooCommerce integration unavailable or disabled.")),
		),
	}
}

// bearerSecurityRequirements builds bearer-auth operation security requirements.
func bearerSecurityRequirements() *openapi3.SecurityRequirements {
	return openapi3.NewSecurityRequirements().With(openapi3.NewSecurityRequirement().Authenticate(bearerSecurityScheme))
}

// responseWithDescription builds OpenAPI responses from plain descriptions.
func responseWithDescription(description string) *openapi3.ResponseRef {
	return &openapi3.ResponseRef{
		Value: openapi3.NewResponse().WithDescription(description),
	}
}

// queryParameter builds optional query-parameter OpenAPI definitions.
func queryParameter(name, description string, schema *openapi3.Schema) *openapi3.ParameterRef {
	return &openapi3.ParameterRef{
		Value: openapi3.NewQueryParameter(name).
			WithDescription(description).
			WithSchema(schema),
	}
}
