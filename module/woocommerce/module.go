package woocommerce

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"go.uber.org/zap"
	contactapplication "mannaiah/module/contacts/application"
	corecron "mannaiah/module/core/cron"
	corehttp "mannaiah/module/core/http"
	contactsadapter "mannaiah/module/woocommerce/adapter/contacts"
	"mannaiah/module/woocommerce/adapter/http"
	wooadapter "mannaiah/module/woocommerce/adapter/woocommerce"
	woocontact "mannaiah/module/woocommerce/application/contact"
	"mannaiah/module/woocommerce/port"
)

var (
	// ErrNilContactService is returned when contact service dependencies are nil.
	ErrNilContactService = errors.New("woocommerce contact service must not be nil")
	// ErrNilSchedulerWhenEnabled is returned when sync is enabled without scheduler dependencies.
	ErrNilSchedulerWhenEnabled = errors.New("woocommerce scheduler must not be nil when sync is enabled")
	// ErrModuleNotInitialized is returned when module startup methods are called on nil receivers.
	ErrModuleNotInitialized = errors.New("woocommerce module is not initialized")
)

// Module defines composition-root wiring for WooCommerce endpoints and schedulers.
type Module struct {
	// cfg defines WooCommerce integration config values.
	cfg Config
	// contactsSyncService defines contact sync use-case dependencies.
	contactsSyncService woocontact.Service
	// handler defines HTTP route adapter dependencies.
	handler *http.Handler
	// scheduler defines optional cron scheduler dependencies.
	scheduler corecron.Scheduler
	// schedulerEntryID defines optional scheduled sync entry identifiers.
	schedulerEntryID corecron.EntryID
	// logger defines structured logging dependencies.
	logger *zap.Logger
	// mutex guards scheduler lifecycle state.
	mutex sync.Mutex
	// started reports whether scheduler lifecycle start logic has completed.
	started bool
}

// Loader defines bootstrap hooks required by WooCommerce modules.
type Loader interface {
	// RegisterRoutes registers module route handlers.
	RegisterRoutes(register func(router corehttp.Router))
	// AddOpenAPISpec merges module OpenAPI specs.
	AddOpenAPISpec(spec *openapi3.T) error
}

// New creates WooCommerce modules with sync service, adapters, and route handlers.
func New(cfg Config, contactService contactapplication.Service, scheduler corecron.Scheduler, providedLogger *zap.Logger, publishers ...port.IntegrationEventPublisher) (*Module, error) {
	if contactService == nil {
		return nil, ErrNilContactService
	}
	if cfg.SyncContacts && scheduler == nil {
		return nil, ErrNilSchedulerWhenEnabled
	}

	logger := resolveLogger(providedLogger)
	upserter, err := contactsadapter.NewUpserter(contactService)
	if err != nil {
		return nil, err
	}

	source, sourceErr := newSource(cfg)
	if sourceErr != nil {
		logger.Warn(
			"woocommerce integration configuration is invalid; endpoint will remain documented and return 503 until fixed",
			zap.Error(sourceErr),
		)
		source = failingSource{err: sourceErr}
	}

	contactSyncService, err := woocontact.NewService(
		woocontact.SyncConfig{
			Enabled:     cfg.SyncContacts,
			PageSize:    cfg.SyncPageSize,
			WorkerCount: cfg.SyncWorkers,
		},
		source,
		upserter,
		resolvePublisher(publishers),
		logger,
	)
	if err != nil {
		return nil, err
	}

	handler, err := http.NewHandler(contactSyncService)
	if err != nil {
		return nil, err
	}

	return &Module{
		cfg:                 cfg,
		contactsSyncService: contactSyncService,
		handler:             handler,
		scheduler:           scheduler,
		logger:              logger,
	}, nil
}

// RegisterRoutes registers WooCommerce routes on the provided router.
func (m *Module) RegisterRoutes(router corehttp.Router) {
	if m == nil || m.handler == nil {
		return
	}

	m.handler.RegisterRoutes(router)
}

// SetAuthorizer configures endpoint authentication and permission dependencies.
func (m *Module) SetAuthorizer(authorizer http.Authorizer) {
	if m == nil || m.handler == nil {
		return
	}

	m.handler.SetAuthorizer(authorizer)
}

// OpenAPISpec returns WooCommerce module OpenAPI documentation.
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

// Start runs startup checks and cron scheduler registration.
func (m *Module) Start(ctx context.Context) error {
	if m == nil {
		return ErrModuleNotInitialized
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.started {
		return nil
	}

	m.validateAtStartup(resolveContext(ctx))
	if !m.cfg.SyncContacts {
		m.started = true
		return nil
	}

	entryID, err := m.scheduler.AddFunc(strings.TrimSpace(m.cfg.SyncContactsCron), func() {
		syncCtx, cancel := context.WithTimeout(context.Background(), resolveValidationTimeout(m.cfg.ValidationTimeoutMS))
		defer cancel()

		if _, syncErr := m.contactsSyncService.SyncContacts(syncCtx, "cron"); syncErr != nil {
			m.logger.Warn("woocommerce cron contacts sync failed", zap.Error(syncErr))
		}
	})
	if err != nil {
		return fmt.Errorf("register woocommerce contacts sync cron: %w", err)
	}

	m.schedulerEntryID = entryID
	m.scheduler.Start()
	m.started = true
	return nil
}

// Stop stops cron scheduling and removes registered jobs.
func (m *Module) Stop(ctx context.Context) error {
	if m == nil {
		return nil
	}

	m.mutex.Lock()
	if !m.started {
		m.mutex.Unlock()
		return nil
	}

	m.started = false
	entryID := m.schedulerEntryID
	m.schedulerEntryID = 0
	scheduler := m.scheduler
	m.mutex.Unlock()

	if scheduler == nil {
		return nil
	}
	if entryID != 0 {
		scheduler.Remove(entryID)
	}
	if err := scheduler.Stop(ctx); err != nil {
		return fmt.Errorf("stop woocommerce scheduler: %w", err)
	}

	return nil
}

// validateAtStartup verifies integration availability and logs startup warnings.
func (m *Module) validateAtStartup(ctx context.Context) {
	validationCtx, cancel := context.WithTimeout(ctx, resolveValidationTimeout(m.cfg.ValidationTimeoutMS))
	defer cancel()

	if err := m.contactsSyncService.ValidateIntegration(validationCtx); err != nil {
		if !errors.Is(err, woocontact.ErrSyncDisabled) {
			m.logger.Warn(
				"woocommerce integration unavailable; endpoints remain documented and return 503 until integration recovers",
				zap.Error(err),
			)
		}
	}
}

// resolveContext resolves nil contexts to background defaults.
func resolveContext(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}

	return context.Background()
}

// resolvePublisher resolves optional integration event publisher dependencies.
func resolvePublisher(publishers []port.IntegrationEventPublisher) port.IntegrationEventPublisher {
	if len(publishers) == 0 {
		return nil
	}

	return publishers[0]
}

// resolveLogger resolves nil loggers to no-op defaults.
func resolveLogger(providedLogger *zap.Logger) *zap.Logger {
	if providedLogger != nil {
		return providedLogger
	}

	return zap.NewNop()
}

// resolveValidationTimeout resolves startup validation timeout values.
func resolveValidationTimeout(timeoutMS int) time.Duration {
	if timeoutMS <= 0 {
		return 3 * time.Second
	}

	return time.Duration(timeoutMS) * time.Millisecond
}

// newSource creates WooCommerce order source adapters from module config values.
func newSource(cfg Config) (port.OrderSource, error) {
	return wooadapter.NewClient(wooadapter.Config{
		URL:            cfg.URL,
		ConsumerKey:    cfg.ConsumerKey,
		ConsumerSecret: cfg.ConsumerSecret,
		Timeout:        time.Duration(resolveRequestTimeout(cfg.RequestTimeoutMS)) * time.Millisecond,
		VerifySSL:      cfg.VerifySSL,
	})
}

// resolveRequestTimeout resolves WooCommerce request timeout values in milliseconds.
func resolveRequestTimeout(timeoutMS int) int {
	if timeoutMS <= 0 {
		return 5000
	}

	return timeoutMS
}

// failingSource defines unavailable WooCommerce source behavior.
type failingSource struct {
	// err defines startup validation errors.
	err error
}

// Validate returns startup validation failures.
func (f failingSource) Validate(ctx context.Context) error {
	return f.err
}

// ListOrders returns startup validation failures.
func (f failingSource) ListOrders(ctx context.Context, page int, pageSize int) (orders []port.WooOrder, hasNext bool, err error) {
	return nil, false, f.err
}
