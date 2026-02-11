package contacts

import (
	"github.com/getkin/kin-openapi/openapi3"
	"gorm.io/gorm"
	"mannaiah/module/contacts/port"
	contactsruntime "mannaiah/module/contacts/runtime"
	corehttp "mannaiah/module/core/http"
)

// Module defines composition-root wiring for contact endpoints.
type Module = contactsruntime.Module

// Loader defines bootstrap hooks required by contacts modules.
type Loader interface {
	// RegisterRoutes registers module route handlers.
	RegisterRoutes(register func(router corehttp.Router))
	// AddOpenAPISpec merges module OpenAPI specs.
	AddOpenAPISpec(spec *openapi3.T) error
}

// New creates a contacts module with schema migration and adapter wiring.
func New(db *gorm.DB, publishers ...port.IntegrationEventPublisher) (*Module, error) {
	return contactsruntime.New(db, publishers...)
}

// OpenAPISpec returns contact-module OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	return contactsruntime.OpenAPISpec()
}
