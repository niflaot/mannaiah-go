package runtime

import (
	"github.com/getkin/kin-openapi/openapi3"
	storefronthttp "mannaiah/module/storefront/adapter/http"
)

// OpenAPISpec returns the aggregated OpenAPI specification for the storefront module.
func OpenAPISpec() *openapi3.T {
	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:   "Storefront API",
			Version: "1.4.0",
		},
		Paths: storefronthttp.Paths(),
		Components: &openapi3.Components{
			SecuritySchemes: storefronthttp.SecuritySchemes(),
			Schemas:         storefronthttp.Schemas(),
		},
		Tags: storefronthttp.Tags(),
	}
}
