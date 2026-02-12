package runtime

import (
	"errors"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"
	"go.uber.org/zap"
	contactapplication "mannaiah/module/contacts/application"
	corecron "mannaiah/module/core/cron"
	corehttp "mannaiah/module/core/http"
	contactsadapter "mannaiah/module/woocommerce/adapter/contacts"
	"mannaiah/module/woocommerce/adapter/http"
	woocontactservice "mannaiah/module/woocommerce/application/contact/service"
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
	contactsSyncService woocontactservice.Service
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

	contactSyncService, err := woocontactservice.NewService(
		woocontactservice.SyncConfig{
			Enabled:     cfg.SyncContacts,
			PageSize:    cfg.SyncPageSize,
			WorkerCount: cfg.SyncWorkers,
		},
		source,
		upserter,
		resolvePublisher(publishers),
		logger,
		woocontactservice.CircuitBreakers{
			Source: newSourceCircuitBreaker(cfg, logger),
		},
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
