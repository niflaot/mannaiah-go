package runtime

import "github.com/getkin/kin-openapi/openapi3"

const (
	// shippingTag defines OpenAPI tags used by shipping endpoints.
	shippingTag = "shipping"
	// bearerSecurityScheme defines OpenAPI security scheme keys used for bearer auth.
	bearerSecurityScheme = "shipping_bearer"
)

// OpenAPISpec returns shipping-module OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	components := openapi3.NewComponents()
	components.SecuritySchemes = openapi3.SecuritySchemes{
		bearerSecurityScheme: &openapi3.SecuritySchemeRef{Value: openapi3.NewJWTSecurityScheme()},
	}
	components.Schemas = openapi3.Schemas{
		"ShippingQuoteRequest":  &openapi3.SchemaRef{Value: shippingQuoteRequestSchema()},
		"ShippingQuoteUnit":     &openapi3.SchemaRef{Value: shippingQuoteUnitSchema()},
		"ShippingQuoteResponse": &openapi3.SchemaRef{Value: shippingQuoteResponseSchema()},
		"ShippingErrorResponse": &openapi3.SchemaRef{Value: shippingErrorResponseSchema()},
	}

	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:   "Shipping API",
			Version: "1.0.0",
		},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/shipping/quotes", &openapi3.PathItem{
				Post: quoteOperation(),
			}),
		),
		Components: &components,
		Tags: openapi3.Tags{
			&openapi3.Tag{Name: shippingTag},
		},
	}
}

// quoteOperation defines the OpenAPI operation for shipping quotes.
func quoteOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShippingQuoteController_quote",
		Summary:     "Create one shipping rate quote",
		Tags:        []string{shippingTag},
		Security:    bearerSecurityRequirements(),
		RequestBody: jsonRequestBodyRef("#/components/schemas/ShippingQuoteRequest"),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithSchemaRef("Quote generated successfully.", "#/components/schemas/ShippingQuoteResponse")),
			openapi3.WithStatus(400, responseWithSchemaRef("Invalid quote request.", "#/components/schemas/ShippingErrorResponse")),
			openapi3.WithStatus(401, responseWithSchemaRef("Unauthorized.", "#/components/schemas/ShippingErrorResponse")),
			openapi3.WithStatus(403, responseWithSchemaRef("Forbidden - Insufficient permissions.", "#/components/schemas/ShippingErrorResponse")),
			openapi3.WithStatus(502, responseWithSchemaRef("Carrier rejected quote request.", "#/components/schemas/ShippingErrorResponse")),
			openapi3.WithStatus(503, responseWithSchemaRef("Shipping integration unavailable.", "#/components/schemas/ShippingErrorResponse")),
		),
	}
}

// shippingQuoteRequestSchema defines OpenAPI schema values for quote request payloads.
func shippingQuoteRequestSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("carrier", openapi3.NewStringSchema().WithEnum("tcc")).
		WithProperty("businessUnit", openapi3.NewStringSchema().WithEnum("courier", "locals")).
		WithProperty("originCityCode", openapi3.NewStringSchema().WithMinLength(5).WithMaxLength(8)).
		WithProperty("destinationCityCode", openapi3.NewStringSchema().WithMinLength(5).WithMaxLength(8)).
		WithProperty("declaredValue", openapi3.NewFloat64Schema()).
		WithProperty("units", openapi3.NewArraySchema().WithItems(shippingQuoteUnitSchema())).
		WithRequired([]string{"carrier", "businessUnit", "originCityCode", "destinationCityCode", "declaredValue", "units"})
}

// shippingQuoteUnitSchema defines OpenAPI schema values for quote unit payloads.
func shippingQuoteUnitSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("number", openapi3.NewIntegerSchema().WithMin(1)).
		WithProperty("realWeight", openapi3.NewFloat64Schema().WithMin(0.000001)).
		WithProperty("height", openapi3.NewFloat64Schema().WithMin(0.000001)).
		WithProperty("width", openapi3.NewFloat64Schema().WithMin(0.000001)).
		WithProperty("length", openapi3.NewFloat64Schema().WithMin(0.000001)).
		WithRequired([]string{"number", "realWeight", "height", "width", "length"})
}

// shippingQuoteResponseSchema defines OpenAPI schema values for quote responses.
func shippingQuoteResponseSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("carrierMessage", openapi3.NewStringSchema()).
		WithProperty("quoteValue", openapi3.NewFloat64Schema()).
		WithProperty("businessUnit", openapi3.NewStringSchema().WithEnum("COURIER", "LOCALS")).
		WithRequired([]string{"carrierMessage", "quoteValue", "businessUnit"})
}

// shippingErrorResponseSchema defines OpenAPI schema values for error responses.
func shippingErrorResponseSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("message", openapi3.NewStringSchema()).
		WithProperty("error", openapi3.NewStringSchema()).
		WithRequired([]string{"message", "error"})
}

// jsonRequestBodyRef builds JSON request body references.
func jsonRequestBodyRef(ref string) *openapi3.RequestBodyRef {
	return &openapi3.RequestBodyRef{
		Value: openapi3.NewRequestBody().WithRequired(true).WithJSONSchemaRef(openapi3.NewSchemaRef(ref, nil)),
	}
}

// responseWithSchemaRef builds OpenAPI responses with JSON schema refs.
func responseWithSchemaRef(description string, ref string) *openapi3.ResponseRef {
	response := openapi3.NewResponse().WithDescription(description)
	response.WithJSONSchemaRef(openapi3.NewSchemaRef(ref, nil))

	return &openapi3.ResponseRef{Value: response}
}

// bearerSecurityRequirements builds bearer-auth operation security requirements.
func bearerSecurityRequirements() *openapi3.SecurityRequirements {
	return openapi3.NewSecurityRequirements().With(openapi3.NewSecurityRequirement().Authenticate(bearerSecurityScheme))
}
