package runtime

import (
	"github.com/getkin/kin-openapi/openapi3"
	"gorm.io/gorm"
	"mannaiah/module/exports/adapter/contacts"
	exportshttp "mannaiah/module/exports/adapter/http"
	"mannaiah/module/exports/adapter/orders"
	"mannaiah/module/exports/adapter/store"
	"mannaiah/module/exports/application"
	"mannaiah/module/exports/port"

	contactsapplication "mannaiah/module/contacts/application"
	corehttp "mannaiah/module/core/http"
	ordersapplication "mannaiah/module/orders/application"
)

// Module defines composition-root wiring for export endpoints.
type Module struct {
	// handler defines HTTP adapter used for route registration.
	handler *exportshttp.Handler
	// service defines application service dependencies for module integrations.
	service application.Service
}

// Loader defines bootstrap hooks required by export modules.
type Loader interface {
	// RegisterRoutes registers module route handlers.
	RegisterRoutes(register func(router corehttp.Router))
	// AddOpenAPISpec merges module OpenAPI specs.
	AddOpenAPISpec(spec *openapi3.T) error
}

// New creates an exports module with adapter wiring.
func New(db *gorm.DB, objectStorage port.Storage, contactsService contactsapplication.Service, ordersService ordersapplication.Service) (*Module, error) {
	repository, err := store.NewRepository(db)
	if err != nil {
		return nil, err
	}
	contactSource, err := contacts.NewSource(contactsService)
	if err != nil {
		return nil, err
	}
	orderSource, err := orders.NewSource(ordersService, contactsService)
	if err != nil {
		return nil, err
	}
	service, err := application.NewService(repository, objectStorage, contactSource, orderSource)
	if err != nil {
		return nil, err
	}
	handler, err := exportshttp.NewHandler(service)
	if err != nil {
		return nil, err
	}

	return &Module{handler: handler, service: service}, nil
}

// RegisterRoutes registers export routes on the provided router.
func (m *Module) RegisterRoutes(router corehttp.Router) {
	if m == nil || m.handler == nil {
		return
	}

	m.handler.RegisterRoutes(router)
}

// Service returns export application service dependencies for module integrations.
func (m *Module) Service() application.Service {
	if m == nil {
		return nil
	}

	return m.service
}

// SetAuthorizer configures endpoint authentication and permission dependencies.
func (m *Module) SetAuthorizer(authorizer exportshttp.Authorizer) {
	if m == nil || m.handler == nil {
		return
	}

	m.handler.SetAuthorizer(authorizer)
}

// OpenAPISpec returns export-module OpenAPI documentation.
func (m *Module) OpenAPISpec() *openapi3.T {
	return OpenAPISpec()
}

// Load mounts all module routes/specs into the provided startup loader.
func (m *Module) Load(loader Loader) error {
	if m == nil || loader == nil {
		return nil
	}

	loader.RegisterRoutes(m.RegisterRoutes)
	if err := loader.AddOpenAPISpec(m.OpenAPISpec()); err != nil {
		return err
	}

	return nil
}
