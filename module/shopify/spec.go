package shopify

import (
	"github.com/getkin/kin-openapi/openapi3"
	shopifyruntime "mannaiah/module/shopify/runtime"
)

// OpenAPISpec returns Shopify module OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	return shopifyruntime.OpenAPISpec()
}
