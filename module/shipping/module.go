package shipping

import (
	"github.com/getkin/kin-openapi/openapi3"
	"gorm.io/gorm"
	shippingruntime "mannaiah/module/shipping/runtime"
)

// Module defines composition-root wiring for shipping endpoints.
type Module = shippingruntime.Module

// Loader defines bootstrap hooks required by shipping modules.
type Loader = shippingruntime.Loader

// New creates shipping modules with adapter wiring.
func New(cfg Config, db *gorm.DB, publishers ...IntegrationEventPublisher) (*Module, error) {
	return shippingruntime.New(cfg, db, publishers...)
}

// OpenAPISpec returns shipping OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	return shippingruntime.OpenAPISpec()
}
