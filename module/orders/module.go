package orders

import (
	"github.com/getkin/kin-openapi/openapi3"
	"gorm.io/gorm"
	corehttp "mannaiah/module/core/http"
	ordersport "mannaiah/module/orders/port"
	ordersruntime "mannaiah/module/orders/runtime"
)

// Module defines composition-root wiring for order endpoints.
type Module = ordersruntime.Module

// Loader defines bootstrap hooks required by order modules.
type Loader interface {
	// RegisterRoutes registers module route handlers.
	RegisterRoutes(register func(router corehttp.Router))
	// AddOpenAPISpec merges module OpenAPI specs.
	AddOpenAPISpec(spec *openapi3.T) error
}

// New creates an orders module with schema migration and adapter wiring.
func New(
	db *gorm.DB,
	customerSource ordersport.CustomerSource,
	resolvers ...ordersport.ProductResolver,
) (*Module, error) {
	return ordersruntime.New(db, customerSource, nil, resolvers...)
}

// NewWithPublisher creates an orders module with schema migration, adapter wiring, and integration event publishing.
func NewWithPublisher(
	db *gorm.DB,
	customerSource ordersport.CustomerSource,
	publisher ordersport.IntegrationEventPublisher,
	resolvers ...ordersport.ProductResolver,
) (*Module, error) {
	return ordersruntime.New(db, customerSource, publisher, resolvers...)
}

// OpenAPISpec returns order-module OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	return ordersruntime.OpenAPISpec()
}
