package runtime

import (
	"github.com/getkin/kin-openapi/openapi3"
	"gorm.io/gorm"
	campaignhttp "mannaiah/module/campaign/adapter/http"
	campaignstore "mannaiah/module/campaign/adapter/store"
	"mannaiah/module/campaign/application"
	"mannaiah/module/campaign/port"
	corehttp "mannaiah/module/core/http"
)

// Loader defines bootstrap hooks required by campaign modules.
type Loader interface {
	// RegisterRoutes registers module route handlers.
	RegisterRoutes(register func(router corehttp.Router))
	// AddOpenAPISpec merges module OpenAPI specs.
	AddOpenAPISpec(spec *openapi3.T) error
}

// Module defines composition-root wiring for campaign endpoints.
type Module struct {
	// cfg defines runtime configuration values.
	cfg Config
	// service defines campaign use-case dependencies.
	service *application.CampaignService
	// handler defines HTTP route adapter dependencies.
	handler *campaignhttp.Handler
}

// New creates campaign modules with adapter wiring.
func New(cfg Config, db *gorm.DB, resolver port.SegmentResolver, sender port.EmailSender) (*Module, error) {
	repository, err := campaignstore.NewRepository(db)
	if err != nil {
		return nil, err
	}

	service, err := application.NewService(repository, resolver, sender, cfg.SendWorkers)
	if err != nil {
		return nil, err
	}

	handler, err := campaignhttp.NewHandler(service)
	if err != nil {
		return nil, err
	}

	return &Module{cfg: cfg, service: service, handler: handler}, nil
}

// RegisterRoutes registers campaign routes on the provided router.
func (m *Module) RegisterRoutes(router corehttp.Router) {
	if m == nil || m.handler == nil || !m.cfg.Enabled {
		return
	}

	m.handler.RegisterRoutes(router)
}

// SetAuthorizer configures endpoint authentication and permission dependencies.
func (m *Module) SetAuthorizer(authorizer campaignhttp.Authorizer) {
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

// Service returns campaign application service dependencies.
func (m *Module) Service() *application.CampaignService {
	if m == nil {
		return nil
	}

	return m.service
}

// OpenAPISpec returns campaign-module OpenAPI documentation.
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
