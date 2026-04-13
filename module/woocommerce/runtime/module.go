package runtime

import (
	"errors"
	"sync"

	contactapplication "mannaiah/module/contacts/application"
	corecron "mannaiah/module/core/cron"
	corehttp "mannaiah/module/core/http"
	"mannaiah/module/core/messaging/bus"
	ordersapplication "mannaiah/module/orders/application"
	contactsadapter "mannaiah/module/woocommerce/adapter/contacts"
	"mannaiah/module/woocommerce/adapter/http"
	woomessaging "mannaiah/module/woocommerce/adapter/messaging"
	ordersadapter "mannaiah/module/woocommerce/adapter/orders"
	woocontactservice "mannaiah/module/woocommerce/application/contact/service"
	woocouponservice "mannaiah/module/woocommerce/application/coupon/service"
	wooorderservice "mannaiah/module/woocommerce/application/order/service"
	"mannaiah/module/woocommerce/port"

	"github.com/getkin/kin-openapi/openapi3"
	"go.uber.org/zap"
)

var (
	// ErrNilContactService is returned when contact service dependencies are nil.
	ErrNilContactService = errors.New("woocommerce contact service must not be nil")
	// ErrNilOrderService is returned when order service dependencies are nil.
	ErrNilOrderService = errors.New("woocommerce order service must not be nil")
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
	// ordersSyncService defines order sync use-case dependencies.
	ordersSyncService wooorderservice.Service
	// couponsSyncService defines optional coupon sync use-case dependencies.
	couponsSyncService woocouponservice.Service
	// handler defines HTTP route adapter dependencies.
	handler *http.Handler
	// scheduler defines optional cron scheduler dependencies.
	scheduler corecron.Scheduler
	// schedulerEntryID defines optional scheduled sync entry identifiers.
	contactsSchedulerEntryID corecron.EntryID
	// ordersSchedulerEntryID defines optional scheduled order-sync entry identifiers.
	ordersSchedulerEntryID corecron.EntryID
	// couponsSchedulerEntryID defines optional scheduled coupon-sync entry identifiers.
	couponsSchedulerEntryID corecron.EntryID
	// logger defines structured logging dependencies.
	logger *zap.Logger
	// orderEventConsumer defines optional cross-module order event consumer dependencies.
	orderEventConsumer *woomessaging.OrderConsumer
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
func New(
	cfg Config,
	contactService contactapplication.Service,
	orderService ordersapplication.Service,
	scheduler corecron.Scheduler,
	providedLogger *zap.Logger,
	registrar bus.Registrar,
	publishers ...port.IntegrationEventPublisher,
) (*Module, error) {
	return NewWithCouponTarget(cfg, contactService, orderService, nil, scheduler, providedLogger, registrar, publishers...)
}

// NewWithCouponTarget creates WooCommerce modules with optional coupon sync target wiring.
func NewWithCouponTarget(
	cfg Config,
	contactService contactapplication.Service,
	orderService ordersapplication.Service,
	couponSyncTarget port.CouponSyncTarget,
	scheduler corecron.Scheduler,
	providedLogger *zap.Logger,
	registrar bus.Registrar,
	publishers ...port.IntegrationEventPublisher,
) (*Module, error) {
	if contactService == nil {
		return nil, ErrNilContactService
	}
	if orderService == nil {
		return nil, ErrNilOrderService
	}
	if (cfg.SyncContacts || cfg.SyncOrders || cfg.SyncCoupons) && scheduler == nil {
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

	orderUpserter, err := ordersadapter.NewUpserter(orderService, contactService)
	if err != nil {
		return nil, err
	}
	if couponUsageSyncService, ok := couponSyncTarget.(ordersadapter.CouponUsageSyncService); ok {
		orderUpserter.SetCouponUsageSyncService(couponUsageSyncService)
	}

	orderSyncService, err := wooorderservice.NewService(
		wooorderservice.SyncConfig{
			Enabled:     cfg.SyncOrders,
			PageSize:    cfg.SyncPageSize,
			WorkerCount: cfg.SyncWorkers,
		},
		source,
		orderUpserter,
		resolvePublisher(publishers),
		logger,
		wooorderservice.CircuitBreakers{
			Source: newSourceCircuitBreaker(cfg, logger),
		},
	)
	if err != nil {
		return nil, err
	}

	var orderEventConsumer *woomessaging.OrderConsumer
	if sourceErr == nil && registrar != nil {
		mainstreamUpdateService, serviceErr := wooorderservice.NewMainstreamUpdateService(
			source,
			logger,
			wooorderservice.CircuitBreakers{Source: newSourceCircuitBreaker(cfg, logger)},
		)
		if serviceErr != nil {
			return nil, serviceErr
		}

		orderEventConsumer, err = woomessaging.NewOrderConsumer(mainstreamUpdateService, logger)
		if err != nil {
			return nil, err
		}
		if err := orderEventConsumer.Register(registrar); err != nil {
			return nil, err
		}
	}

	var couponsSyncService woocouponservice.Service
	if couponSyncTarget != nil {
		couponsSyncService, err = woocouponservice.NewService(
			woocouponservice.SyncConfig{
				Enabled:  cfg.SyncCoupons,
				PageSize: cfg.SyncPageSize,
			},
			source,
			couponSyncTarget,
			resolvePublisher(publishers),
			logger,
			woocouponservice.CircuitBreakers{
				Source: newSourceCircuitBreaker(cfg, logger),
			},
		)
		if err != nil {
			return nil, err
		}
	}

	handler, err := http.NewHandler(contactSyncService, orderSyncService)
	if err != nil {
		return nil, err
	}
	handler.SetCouponSyncService(couponsSyncService)

	return &Module{
		cfg:                 cfg,
		contactsSyncService: contactSyncService,
		ordersSyncService:   orderSyncService,
		couponsSyncService:  couponsSyncService,
		handler:             handler,
		scheduler:           scheduler,
		logger:              logger,
		orderEventConsumer:  orderEventConsumer,
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

// SetSyncRecorder configures optional sync run recording dependencies.
func (m *Module) SetSyncRecorder(recorder port.SyncRecorder) {
	if m == nil {
		return
	}

	if contactSyncService, ok := m.contactsSyncService.(*woocontactservice.ContactSyncService); ok {
		contactSyncService.SetSyncRecorder(recorder)
	}
	if orderSyncService, ok := m.ordersSyncService.(*wooorderservice.OrderSyncService); ok {
		orderSyncService.SetSyncRecorder(recorder)
	}
	if couponSyncService, ok := m.couponsSyncService.(*woocouponservice.CouponSyncService); ok {
		couponSyncService.SetSyncRecorder(recorder)
	}
}

// SetMembershipStamper configures optional membership stamp dependencies.
func (m *Module) SetMembershipStamper(stamper port.MembershipStamper) {
	if m == nil {
		return
	}

	if contactSyncService, ok := m.contactsSyncService.(*woocontactservice.ContactSyncService); ok {
		contactSyncService.SetMembershipStamper(stamper)
	}
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
