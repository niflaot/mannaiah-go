package runtime

import (
	"github.com/getkin/kin-openapi/openapi3"
	"gorm.io/gorm"
	analyticsport "mannaiah/module/analytics/port"
	corehttp "mannaiah/module/core/http"
	segmenthttp "mannaiah/module/segment/adapter/http"
	segmentstore "mannaiah/module/segment/adapter/store"
	"mannaiah/module/segment/application"
)

// Loader defines bootstrap hooks required by segment modules.
type Loader interface {
	// RegisterRoutes registers module route handlers.
	RegisterRoutes(register func(router corehttp.Router))
	// AddOpenAPISpec merges module OpenAPI specs.
	AddOpenAPISpec(spec *openapi3.T) error
}

// Module defines composition-root wiring for segment endpoints.
type Module struct {
	// cfg defines runtime configuration values.
	cfg Config
	// service defines application service dependencies.
	service *application.SegmentService
	// handler defines HTTP route adapter dependencies.
	handler *segmenthttp.Handler
}

// New creates segment modules with adapter wiring.
func New(cfg Config, db *gorm.DB, resolver analyticsport.Resolver) (*Module, error) {
	repository, err := segmentstore.NewRepository(db)
	if err != nil {
		return nil, err
	}

	service, err := application.NewService(repository, resolver, db)
	if err != nil {
		return nil, err
	}

	handler, err := segmenthttp.NewHandler(service)
	if err != nil {
		return nil, err
	}

	return &Module{cfg: cfg, service: service, handler: handler}, nil
}

// RegisterRoutes registers segment routes on the provided router.
func (m *Module) RegisterRoutes(router corehttp.Router) {
	if m == nil || m.handler == nil || !m.cfg.Enabled {
		return
	}

	m.handler.RegisterRoutes(router)
}

// SetAuthorizer configures endpoint authentication and permission dependencies.
func (m *Module) SetAuthorizer(authorizer segmenthttp.Authorizer) {
	if m == nil || m.handler == nil {
		return
	}

	m.handler.SetAuthorizer(authorizer)
}

// Service returns segment application service dependencies.
func (m *Module) Service() *application.SegmentService {
	if m == nil {
		return nil
	}

	return m.service
}

// OpenAPISpec returns segment-module OpenAPI documentation.
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
