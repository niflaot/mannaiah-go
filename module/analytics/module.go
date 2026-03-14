package analytics

import (
	"github.com/getkin/kin-openapi/openapi3"
	"gorm.io/gorm"
	analyticsruntime "mannaiah/module/analytics/runtime"
	"mannaiah/module/core/messaging/bus"
)

// Module defines composition-root wiring for analytics endpoints.
type Module = analyticsruntime.Module

// Loader defines bootstrap hooks required by analytics modules.
type Loader = analyticsruntime.Loader

// New creates analytics modules with adapter wiring.
func New(cfg Config, db *gorm.DB, registrar bus.Registrar) (*Module, error) {
	return analyticsruntime.New(cfg, db, registrar)
}

// OpenAPISpec returns analytics OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	return analyticsruntime.OpenAPISpec()
}
