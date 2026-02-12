package assets

import (
	"github.com/getkin/kin-openapi/openapi3"
	"gorm.io/gorm"
	"mannaiah/module/assets/port"
	assetsruntime "mannaiah/module/assets/runtime"
	corehttp "mannaiah/module/core/http"
)

// Module defines composition-root wiring for asset endpoints.
type Module = assetsruntime.Module

// Loader defines bootstrap hooks required by assets modules.
type Loader interface {
	// RegisterRoutes registers module route handlers.
	RegisterRoutes(register func(router corehttp.Router))
	// AddOpenAPISpec merges module OpenAPI specs.
	AddOpenAPISpec(spec *openapi3.T) error
}

// New creates an assets module with schema migration and adapter wiring.
func New(db *gorm.DB, storage port.Storage, publishers ...port.IntegrationEventPublisher) (*Module, error) {
	return assetsruntime.New(db, storage, publishers...)
}

// OpenAPISpec returns assets-module OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	return assetsruntime.OpenAPISpec()
}
