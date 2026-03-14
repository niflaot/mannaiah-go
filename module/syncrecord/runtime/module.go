package runtime

import (
	"context"
	"errors"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	corecron "mannaiah/module/core/cron"
	corehttp "mannaiah/module/core/http"
	syncrecordhttp "mannaiah/module/syncrecord/adapter/http"
	syncrecordstore "mannaiah/module/syncrecord/adapter/store"
	"mannaiah/module/syncrecord/application"
	"mannaiah/module/syncrecord/port"

	"gorm.io/gorm"
)

var (
	// ErrNilDB is returned when nil db dependencies are provided.
	ErrNilDB = errors.New("sync record db must not be nil")
)

// Loader defines bootstrap hooks required by sync record modules.
type Loader interface {
	// RegisterRoutes registers module route handlers.
	RegisterRoutes(register func(router corehttp.Router))
	// AddOpenAPISpec merges module OpenAPI specs.
	AddOpenAPISpec(spec *openapi3.T) error
}

// Module defines composition-root wiring for sync record endpoints.
type Module struct {
	// cfg defines runtime configuration values.
	cfg Config
	// service defines application service dependencies.
	service *application.RecorderService
	// handler defines HTTP route adapter dependencies.
	handler *syncrecordhttp.Handler
	// scheduler defines optional cron scheduler dependencies.
	scheduler corecron.Scheduler
	// cleanupEntryID defines cleanup entry identifiers.
	cleanupEntryID corecron.EntryID
	// started reports lifecycle state.
	started bool
}

// New creates sync record modules with adapter wiring.
func New(cfg Config, db *gorm.DB, schedulers ...corecron.Scheduler) (*Module, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	repo, err := syncrecordstore.NewRepository(db)
	if err != nil {
		return nil, err
	}
	service, err := application.NewService(repo)
	if err != nil {
		return nil, err
	}
	handler, err := syncrecordhttp.NewHandler(service)
	if err != nil {
		return nil, err
	}

	module := &Module{cfg: cfg, service: service, handler: handler}
	if len(schedulers) > 0 {
		module.scheduler = schedulers[0]
	}

	return module, nil
}

// ConfigureScheduler configures cleanup cron scheduler dependencies.
func (m *Module) ConfigureScheduler(scheduler corecron.Scheduler) {
	if m == nil {
		return
	}

	m.scheduler = scheduler
}

// RegisterRoutes registers sync record routes on the provided router.
func (m *Module) RegisterRoutes(router corehttp.Router) {
	if m == nil || m.handler == nil || !m.cfg.Enabled {
		return
	}

	m.handler.RegisterRoutes(router)
}

// SetAuthorizer configures endpoint authentication and permission dependencies.
func (m *Module) SetAuthorizer(authorizer syncrecordhttp.Authorizer) {
	if m == nil || m.handler == nil {
		return
	}

	m.handler.SetAuthorizer(authorizer)
}

// Service returns sync record application service dependencies.
func (m *Module) Service() *application.RecorderService {
	if m == nil {
		return nil
	}

	return m.service
}

// Recorder returns sync record port used by other modules.
func (m *Module) Recorder() port.Recorder {
	if m == nil || m.service == nil || !m.cfg.Enabled {
		return port.NoopRecorder{}
	}

	return m.service
}

// OpenAPISpec returns sync record OpenAPI documentation.
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

// Start starts cleanup cron behavior when enabled.
func (m *Module) Start(_ context.Context) error {
	if m == nil || m.started {
		return nil
	}
	if !m.cfg.Enabled || !m.cfg.CleanupEnabled || m.scheduler == nil {
		m.started = true
		return nil
	}

	entryID, err := m.scheduler.AddFunc(m.cfg.CleanupCron, func() {
		ctx, cancel := context.WithTimeout(context.Background(), resolveTimeout(m.cfg.CleanupTimeoutMS))
		defer cancel()
		_, _ = m.service.CleanupExpired(ctx, m.cfg.RetentionDays)
	})
	if err != nil {
		return err
	}

	m.cleanupEntryID = entryID
	m.scheduler.Start()
	m.started = true
	return nil
}

// Stop stops cleanup cron behavior.
func (m *Module) Stop(ctx context.Context) error {
	if m == nil || !m.started {
		return nil
	}
	m.started = false
	if m.scheduler == nil {
		return nil
	}
	if m.cleanupEntryID != 0 {
		m.scheduler.Remove(m.cleanupEntryID)
		m.cleanupEntryID = 0
	}

	return m.scheduler.Stop(resolveContext(ctx))
}

// resolveTimeout resolves cleanup timeout values.
func resolveTimeout(timeoutMS int) time.Duration {
	if timeoutMS <= 0 {
		return time.Minute
	}

	return time.Duration(timeoutMS) * time.Millisecond
}

// resolveContext resolves nil contexts.
func resolveContext(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}

	return context.Background()
}
