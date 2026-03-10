package runtime

import (
	"errors"
	assethttp "mannaiah/module/assets/adapter/http"
	assetstore "mannaiah/module/assets/adapter/store"
	assetsapplication "mannaiah/module/assets/application"
	"mannaiah/module/assets/port"
	corecron "mannaiah/module/core/cron"
	corehttp "mannaiah/module/core/http"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	// ErrModuleNotInitialized is returned when module startup methods are called on nil receivers.
	ErrModuleNotInitialized = errors.New("assets module is not initialized")
	// ErrNilSchedulerWhenEnabled is returned when JPG worker is enabled without scheduler dependencies.
	ErrNilSchedulerWhenEnabled = errors.New("assets scheduler must not be nil when jpg worker is enabled")
)

// Module defines composition-root wiring for asset endpoints.
type Module struct {
	// cfg defines assets integration config values.
	cfg Config
	// handler defines HTTP adapter used for route registration.
	handler *assethttp.Handler
	// service defines application service dependencies for module integrations.
	service assetsapplication.Service
	// scheduler defines optional cron scheduler dependencies.
	scheduler corecron.Scheduler
	// schedulerEntryID defines optional scheduled worker entry identifiers.
	schedulerEntryID corecron.EntryID
	// logger defines structured logging dependencies.
	logger *zap.Logger
	// mutex guards scheduler lifecycle state.
	mutex sync.Mutex
	// started reports whether scheduler lifecycle start logic has completed.
	started bool
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
	return NewWithConfig(Config{}, db, storage, nil, publishers...)
}

// NewWithConfig creates an assets module with config-driven worker wiring.
func NewWithConfig(cfg Config, db *gorm.DB, storage port.Storage, providedLogger *zap.Logger, publishers ...port.IntegrationEventPublisher) (*Module, error) {
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
	handler.SetJPGWorkerDefaults(assetsapplication.JPGWorkerCommand{
		Tags:        resolveWorkerTags(cfg.JPGWorkerTags),
		BatchSize:   cfg.JPGWorkerBatchSize,
		JPEGQuality: cfg.JPGWorkerQuality,
	})

	return &Module{
		cfg:     cfg,
		handler: handler,
		service: service,
		logger:  resolveLogger(providedLogger),
	}, nil
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

// resolveLogger resolves nil logger dependencies into no-op logger values.
func resolveLogger(providedLogger *zap.Logger) *zap.Logger {
	if providedLogger == nil {
		return zap.NewNop()
	}

	return providedLogger
}
