package woocommerce

import (
	"github.com/getkin/kin-openapi/openapi3"
	wooruntime "mannaiah/module/woocommerce/runtime"
)

// OpenAPISpec returns WooCommerce module OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	return wooruntime.OpenAPISpec()
}
