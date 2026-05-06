package shopify

import (
	shopifyruntime "mannaiah/module/shopify/runtime"

	"github.com/getkin/kin-openapi/openapi3"
)

// OpenAPISpec returns Shopify module OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	return shopifyruntime.OpenAPISpec()
}
