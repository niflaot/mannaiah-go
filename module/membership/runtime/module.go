package runtime

import (
	"github.com/getkin/kin-openapi/openapi3"
	corehttp "mannaiah/module/core/http"
	membershiphttp "mannaiah/module/membership/adapter/http"
	membershipstore "mannaiah/module/membership/adapter/store"
	"mannaiah/module/membership/application"
	"mannaiah/module/membership/port"

	"gorm.io/gorm"
)

// Loader defines bootstrap hooks required by membership modules.
type Loader interface {
	// RegisterRoutes registers module route handlers.
	RegisterRoutes(register func(router corehttp.Router))
	// AddOpenAPISpec merges module OpenAPI specs.
	AddOpenAPISpec(spec *openapi3.T) error
}

// Module defines composition-root wiring for membership endpoints.
type Module struct {
	// cfg defines runtime configuration values.
	cfg Config
	// service defines application service dependencies.
	service *application.MembershipService
	// handler defines HTTP route adapter dependencies.
	handler *membershiphttp.Handler
}

// New creates membership modules with adapter wiring.
func New(cfg Config, db *gorm.DB, contacts port.ContactLookup, publishers ...port.IntegrationEventPublisher) (*Module, error) {
	repository, err := membershipstore.NewRepository(db)
	if err != nil {
		return nil, err
	}

	service, err := application.NewService(repository, contacts, publishers...)
	if err != nil {
		return nil, err
	}

	handler, err := membershiphttp.NewHandler(service)
	if err != nil {
		return nil, err
	}

	return &Module{cfg: cfg, service: service, handler: handler}, nil
}

// RegisterRoutes registers membership routes on the provided router.
func (m *Module) RegisterRoutes(router corehttp.Router) {
	if m == nil || m.handler == nil || !m.cfg.Enabled {
		return
	}

	m.handler.RegisterRoutes(router)
}

// SetAuthorizer configures endpoint authentication and permission dependencies.
func (m *Module) SetAuthorizer(authorizer membershiphttp.Authorizer) {
	if m == nil || m.handler == nil {
		return
	}

	m.handler.SetAuthorizer(authorizer)
}

// SetSyncRecorder configures optional sync run recording dependencies.
func (m *Module) SetSyncRecorder(recorder port.SyncRecorder) {
	if m == nil || m.service == nil {
		return
	}

	m.service.SetSyncRecorder(recorder)
}

// Service returns membership application service dependencies.
func (m *Module) Service() *application.MembershipService {
	if m == nil {
		return nil
	}

	return m.service
}

// OpenAPISpec returns membership-module OpenAPI documentation.
func (m *Module) OpenAPISpec() *openapi3.T {
	return OpenAPISpec()
}

// Load mounts all module routes/specs into the provided startup loader.
func (m *Module) Load(loader Loader) error {
	if m == nil || loader == nil {
		return nil
	}
	if m.cfg.Enabled {
		loader.RegisterRoutes(m.RegisterRoutes)
	}
	if err := loader.AddOpenAPISpec(m.OpenAPISpec()); err != nil {
		return err
	}

	return nil
}
