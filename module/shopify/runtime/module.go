package runtime

import (
	"errors"
	"fmt"

	contactapplication "mannaiah/module/contacts/application"
	corehttp "mannaiah/module/core/http"
	"mannaiah/module/core/messaging/bus"
	ordersapplication "mannaiah/module/orders/application"
	shopifyhttp "mannaiah/module/shopify/adapter/http"
	shopifymessaging "mannaiah/module/shopify/adapter/messaging"
	shopifystore "mannaiah/module/shopify/adapter/store"
	shopifycontactservice "mannaiah/module/shopify/application/contact/service"
	shopifyorderservice "mannaiah/module/shopify/application/order/service"
	shopifyport "mannaiah/module/shopify/port"

	"github.com/getkin/kin-openapi/openapi3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	// ErrNilDB is returned when DB dependencies are nil.
	ErrNilDB = errors.New("shopify db must not be nil")
	// ErrNilContactService is returned when contact service dependencies are nil.
	ErrNilContactService = errors.New("shopify contact service must not be nil")
	// ErrNilOrderService is returned when order service dependencies are nil.
	ErrNilOrderService = errors.New("shopify order service must not be nil")
	// ErrModuleNotInitialized is returned when module lifecycle methods are called on nil receivers.
	ErrModuleNotInitialized = errors.New("shopify module is not initialized")
)

// Module defines composition-root wiring for Shopify endpoints and consumers.
type Module struct {
	cfg Config
	// logger defines structured logging dependencies.
	logger *zap.Logger
	// installationResolver defines cached Shopify installation lookup dependencies.
	installationResolver shopifyport.InstallationResolver
	// handler defines HTTP route handler dependencies.
	handler *shopifyhttp.Handler
	// processor defines asynchronous webhook processing dependencies.
	processor *shopifyhttp.Processor
	// contactSyncService defines targeted contact sync dependencies.
	contactSyncService *shopifycontactservice.ContactSyncService
	// orderSyncService defines targeted order sync dependencies.
	orderSyncService *shopifyorderservice.OrderSyncService
	// contactConsumer defines outbound contact event consumer dependencies.
	contactConsumer *shopifymessaging.ContactConsumer
	// orderConsumer defines outbound order event consumer dependencies.
	orderConsumer *shopifymessaging.OrderConsumer
	// registrar defines optional integration-event registration dependencies.
	registrar bus.Registrar
	// consumerRegistered reports whether integration handlers were registered.
	consumerRegistered bool
}

// Loader defines bootstrap hooks required by Shopify modules.
type Loader interface {
	// RegisterRoutes registers module route handlers.
	RegisterRoutes(register func(router corehttp.Router))
	// AddOpenAPISpec merges module OpenAPI specs.
	AddOpenAPISpec(spec *openapi3.T) error
}

// New creates Shopify modules with DB-backed repositories and direct service wiring.
func New(
	cfg Config,
	db *gorm.DB,
	contactService contactapplication.Service,
	orderService ordersapplication.Service,
	providedLogger *zap.Logger,
	registrar bus.Registrar,
	publishers ...shopifyport.IntegrationEventPublisher,
) (*Module, error) {
	_ = publishers
	if db == nil {
		return nil, ErrNilDB
	}
	if contactService == nil {
		return nil, ErrNilContactService
	}
	if orderService == nil {
		return nil, ErrNilOrderService
	}
	logger := resolveLogger(providedLogger)
	repository, err := shopifystore.NewRepository(db)
	if err != nil {
		return nil, fmt.Errorf("create shopify repository: %w", err)
	}
	installationResolver := newInstallationResolver(repository)

	source, sourceErr := newSource(cfg, installationResolver)
	if sourceErr != nil {
		logger.Warn("shopify source initialization failed; integration will stay unavailable", zap.Error(sourceErr))
		source = failingSource{err: sourceErr}
	}

	contactTarget, err := shopifycontactservice.NewUpserter(contactService, repository, source, logger)
	if err != nil {
		return nil, fmt.Errorf("create shopify contact target: %w", err)
	}
	orderTarget, err := shopifyorderservice.NewUpserter(orderService, repository)
	if err != nil {
		return nil, fmt.Errorf("create shopify order target: %w", err)
	}

	contactSyncService, err := shopifycontactservice.NewService(
		shopifycontactservice.SyncConfig{Enabled: cfg.SyncContacts},
		source,
		contactTarget,
		logger,
		shopifycontactservice.CircuitBreakers{Source: newSourceCircuitBreaker(cfg, logger)},
	)
	if err != nil {
		return nil, fmt.Errorf("create shopify contact sync service: %w", err)
	}
	orderSyncService, err := shopifyorderservice.NewService(
		shopifyorderservice.SyncConfig{Enabled: cfg.SyncOrders, Realm: "shopify"},
		source,
		contactTarget,
		orderTarget,
		logger,
		shopifyorderservice.CircuitBreakers{Source: newSourceCircuitBreaker(cfg, logger)},
	)
	if err != nil {
		return nil, fmt.Errorf("create shopify order sync service: %w", err)
	}

	processor, err := shopifyhttp.NewProcessor(resolveSyncWorkers(cfg.SyncWorkers), resolveSyncTimeout(cfg.SyncTimeoutMS), contactSyncService, orderSyncService, logger)
	if err != nil {
		return nil, fmt.Errorf("create shopify webhook processor: %w", err)
	}
	handler, err := shopifyhttp.NewHandler(
		contactSyncService,
		orderSyncService,
		processor,
		repository,
		repository,
		repository,
		installationResolver,
		source,
		contactService,
		orderService,
		cfg.ClientID,
		cfg.ClientSecret,
	)
	if err != nil {
		return nil, fmt.Errorf("create shopify http handler: %w", err)
	}

	var contactConsumer *shopifymessaging.ContactConsumer
	if cfg.SyncContacts {
		mainstreamContactUpdateService, err := shopifycontactservice.NewMainstreamUpdateService(
			source,
			source,
			repository,
			logger,
			shopifycontactservice.CircuitBreakers{
				Source:      newSourceCircuitBreaker(cfg, logger),
				Destination: newDestinationCircuitBreaker(cfg, logger),
			},
		)
		if err != nil {
			return nil, fmt.Errorf("create shopify mainstream contact update service: %w", err)
		}
		contactConsumer, err = shopifymessaging.NewContactConsumer(mainstreamContactUpdateService, logger)
		if err != nil {
			return nil, fmt.Errorf("create shopify contact consumer: %w", err)
		}
	}

	var orderConsumer *shopifymessaging.OrderConsumer
	if cfg.SyncOrders {
		mainstreamUpdateService, err := shopifyorderservice.NewMainstreamUpdateService(
			source,
			repository,
			logger,
			shopifyorderservice.CircuitBreakers{Destination: newDestinationCircuitBreaker(cfg, logger)},
		)
		if err != nil {
			return nil, fmt.Errorf("create shopify mainstream update service: %w", err)
		}
		orderConsumer, err = shopifymessaging.NewOrderConsumer(mainstreamUpdateService, logger)
		if err != nil {
			return nil, fmt.Errorf("create shopify order consumer: %w", err)
		}
	}

	return &Module{
		cfg:                  cfg,
		logger:               logger,
		installationResolver: installationResolver,
		handler:              handler,
		processor:            processor,
		contactSyncService:   contactSyncService,
		orderSyncService:     orderSyncService,
		contactConsumer:      contactConsumer,
		orderConsumer:        orderConsumer,
		registrar:            registrar,
	}, nil
}

// RegisterRoutes registers Shopify routes on the provided router.
func (m *Module) RegisterRoutes(router corehttp.Router) {
	if m == nil || m.handler == nil || router == nil {
		return
	}

	m.handler.RegisterRoutes(router)
}

// SetAuthorizer configures endpoint authentication dependencies.
func (m *Module) SetAuthorizer(authorizer any) {
	if m == nil || m.handler == nil || authorizer == nil {
		return
	}
	resolved, ok := authorizer.(shopifyhttp.Authorizer)
	if !ok {
		return
	}

	m.handler.SetAuthorizer(resolved)
}

// SetSyncRecorder configures optional sync run recording dependencies.
func (m *Module) SetSyncRecorder(recorder shopifyport.SyncRecorder) {
	if m == nil {
		return
	}
	if m.contactSyncService != nil {
		m.contactSyncService.SetSyncRecorder(recorder)
	}
	if m.orderSyncService != nil {
		m.orderSyncService.SetSyncRecorder(recorder)
	}
}

// OpenAPISpec returns Shopify module OpenAPI documentation.
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
