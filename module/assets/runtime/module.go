package runtime

import (
	assethttp "mannaiah/module/assets/adapter/http"
	assetstore "mannaiah/module/assets/adapter/store"
	assetsapplication "mannaiah/module/assets/application"
	"mannaiah/module/assets/port"
	corehttp "mannaiah/module/core/http"

	"github.com/getkin/kin-openapi/openapi3"
	"gorm.io/gorm"
)

// Module defines composition-root wiring for asset endpoints.
type Module struct {
	// handler defines HTTP adapter used for route registration.
	handler *assethttp.Handler
	// service defines application service dependencies for module integrations.
	service assetsapplication.Service
}

// Loader defines bootstrap hooks required by assets modules.
type Loader interface {
	// RegisterRoutes registers module route handlers.
	RegisterRoutes(register func(router corehttp.Router))
	// AddOpenAPISpec merges module OpenAPI specs.
	AddOpenAPISpec(spec *openapi3.T) error
}

// New creates an assets module with adapter wiring.
func New(db *gorm.DB, storage port.Storage, publishers ...port.IntegrationEventPublisher) (*Module, error) {
	repository, err := assetstore.NewRepository(db)
	if err != nil {
		return nil, err
	}

	service, err := assetsapplication.NewService(repository, storage, resolvePublisher(publishers))
	if err != nil {
		return nil, err
	}

	handler, err := assethttp.NewHandler(service)
	if err != nil {
		return nil, err
	}

	return &Module{handler: handler, service: service}, nil
}

// RegisterRoutes registers asset routes on the provided router.
func (m *Module) RegisterRoutes(router corehttp.Router) {
	if m == nil || m.handler == nil {
		return
	}

	m.handler.RegisterRoutes(router)
}

// Service returns asset application service dependencies for module integrations.
func (m *Module) Service() assetsapplication.Service {
	if m == nil {
		return nil
	}

	return m.service
}

// SetAuthorizer configures endpoint authentication and permission dependencies.
func (m *Module) SetAuthorizer(authorizer assethttp.Authorizer) {
	if m == nil || m.handler == nil {
		return
	}

	m.handler.SetAuthorizer(authorizer)
}

// OpenAPISpec returns assets-module OpenAPI documentation.
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
