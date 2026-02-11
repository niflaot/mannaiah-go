package runtime

import (
	"context"
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	"gorm.io/gorm"
	"mannaiah/module/contacts/adapter/http"
	"mannaiah/module/contacts/adapter/store"
	"mannaiah/module/contacts/application"
	"mannaiah/module/contacts/port"
	corehttp "mannaiah/module/core/http"
)

// Module defines composition-root wiring for contact endpoints.
type Module struct {
	// handler defines HTTP adapter used for route registration.
	handler *http.Handler
	// service defines application service dependencies for module integrations.
	service application.Service
}

// Loader defines bootstrap hooks required by contacts modules.
type Loader interface {
	// RegisterRoutes registers module route handlers.
	RegisterRoutes(register func(router corehttp.Router))
	// AddOpenAPISpec merges module OpenAPI specs.
	AddOpenAPISpec(spec *openapi3.T) error
}

// New creates a contacts module with schema migration and adapter wiring.
func New(db *gorm.DB, publishers ...port.IntegrationEventPublisher) (*Module, error) {
	repository, err := store.NewRepository(db)
	if err != nil {
		return nil, err
	}
	if err := repository.EnsureSchema(context.Background()); err != nil {
		return nil, fmt.Errorf("ensure contacts schema: %w", err)
	}

	service, err := application.NewServiceWithPublisher(repository, resolvePublisher(publishers))
	if err != nil {
		return nil, err
	}

	handler, err := http.NewHandler(service)
	if err != nil {
		return nil, err
	}

	return &Module{handler: handler, service: service}, nil
}

// RegisterRoutes registers contacts routes on the provided router.
func (m *Module) RegisterRoutes(router corehttp.Router) {
	if m == nil || m.handler == nil {
		return
	}

	m.handler.RegisterRoutes(router)
}

// Service returns contact application service dependencies for module integrations.
func (m *Module) Service() application.Service {
	if m == nil {
		return nil
	}

	return m.service
}

// SetAuthorizer configures endpoint authentication and permission dependencies.
func (m *Module) SetAuthorizer(authorizer http.Authorizer) {
	if m == nil || m.handler == nil {
		return
	}

	m.handler.SetAuthorizer(authorizer)
}

// OpenAPISpec returns contact-module OpenAPI documentation.
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

// resolvePublisher resolves optional integration event publisher dependencies.
func resolvePublisher(publishers []port.IntegrationEventPublisher) port.IntegrationEventPublisher {
	if len(publishers) == 0 {
		return nil
	}

	return publishers[0]
}
