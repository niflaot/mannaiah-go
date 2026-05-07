package exports

import (
	"github.com/getkin/kin-openapi/openapi3"
	"gorm.io/gorm"
	contactsapplication "mannaiah/module/contacts/application"
	corehttp "mannaiah/module/core/http"
	ordersapplication "mannaiah/module/orders/application"

	exportsport "mannaiah/module/exports/port"
	exportsruntime "mannaiah/module/exports/runtime"
)

// Module defines composition-root wiring for export endpoints.
type Module = exportsruntime.Module

// Loader defines bootstrap hooks required by export modules.
type Loader interface {
	// RegisterRoutes registers module route handlers.
	RegisterRoutes(register func(router corehttp.Router))
	// AddOpenAPISpec merges module OpenAPI specs.
	AddOpenAPISpec(spec *openapi3.T) error
}

// New creates an exports module with schema migration and adapter wiring.
func New(db *gorm.DB, storage exportsport.Storage, contacts contactsapplication.Service, orders ordersapplication.Service) (*Module, error) {
	return exportsruntime.New(db, storage, contacts, orders)
}

// OpenAPISpec returns export-module OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	return exportsruntime.OpenAPISpec()
}
