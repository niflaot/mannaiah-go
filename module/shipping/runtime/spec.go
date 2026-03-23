package runtime

import (
	"github.com/getkin/kin-openapi/openapi3"

	shippinghttp "mannaiah/module/shipping/adapter/http"
)

// OpenAPISpec returns shipping-module OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	components := openapi3.NewComponents()
	components.SecuritySchemes = shippinghttp.SecuritySchemes()

	return &openapi3.T{
		OpenAPI:    "3.0.3",
		Info:       &openapi3.Info{Title: "Shipping API", Version: "1.0.0"},
		Paths:      shippinghttp.Paths(),
		Components: &components,
		Tags:       shippinghttp.Tags(),
	}
}
