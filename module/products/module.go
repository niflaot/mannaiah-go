package products

import (
	"github.com/getkin/kin-openapi/openapi3"
	"gorm.io/gorm"
	corehttp "mannaiah/module/core/http"
	productsruntime "mannaiah/module/products/runtime"
)

// Module defines composition-root wiring for product endpoints.
type Module = productsruntime.Module

// Loader defines bootstrap hooks required by products modules.
type Loader interface {
	// RegisterRoutes registers module route handlers.
	RegisterRoutes(register func(router corehttp.Router))
	// AddOpenAPISpec merges module OpenAPI specs.
	AddOpenAPISpec(spec *openapi3.T) error
}

// New creates a products module with schema migration and adapter wiring.
func New(db *gorm.DB) (*Module, error) {
	return productsruntime.New(db)
}

// OpenAPISpec returns product-module OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	return productsruntime.OpenAPISpec()
}
