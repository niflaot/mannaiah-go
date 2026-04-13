package woocommerce

import (
	contactapplication "mannaiah/module/contacts/application"
	corecron "mannaiah/module/core/cron"
	"mannaiah/module/core/messaging/bus"
	ordersapplication "mannaiah/module/orders/application"
	"mannaiah/module/woocommerce/port"
	wooruntime "mannaiah/module/woocommerce/runtime"

	"go.uber.org/zap"
)

var (
	// ErrNilContactService is returned when contact service dependencies are nil.
	ErrNilContactService = wooruntime.ErrNilContactService
	// ErrNilOrderService is returned when order service dependencies are nil.
	ErrNilOrderService = wooruntime.ErrNilOrderService
	// ErrNilSchedulerWhenEnabled is returned when sync is enabled without scheduler dependencies.
	ErrNilSchedulerWhenEnabled = wooruntime.ErrNilSchedulerWhenEnabled
	// ErrModuleNotInitialized is returned when module startup methods are called on nil receivers.
	ErrModuleNotInitialized = wooruntime.ErrModuleNotInitialized
)

// Module defines composition-root wiring for WooCommerce endpoints and schedulers.
type Module = wooruntime.Module

// Loader defines bootstrap hooks required by WooCommerce modules.
type Loader = wooruntime.Loader

// New creates WooCommerce modules with sync service, adapters, and route handlers.
func New(
	cfg Config,
	contactService contactapplication.Service,
	orderService ordersapplication.Service,
	scheduler corecron.Scheduler,
	providedLogger *zap.Logger,
	publishers ...port.IntegrationEventPublisher,
) (*Module, error) {
	return wooruntime.New(cfg, contactService, orderService, scheduler, providedLogger, nil, publishers...)
}

// NewWithMessaging creates WooCommerce modules with messaging subscriptions for cross-module integrations.
func NewWithMessaging(
	cfg Config,
	contactService contactapplication.Service,
	orderService ordersapplication.Service,
	scheduler corecron.Scheduler,
	providedLogger *zap.Logger,
	registrar bus.Registrar,
	publishers ...port.IntegrationEventPublisher,
) (*Module, error) {
	return wooruntime.New(cfg, contactService, orderService, scheduler, providedLogger, registrar, publishers...)
}

// NewWithMessagingAndCouponTarget creates WooCommerce modules with messaging and coupon sync target wiring.
func NewWithMessagingAndCouponTarget(
	cfg Config,
	contactService contactapplication.Service,
	orderService ordersapplication.Service,
	couponSyncTarget port.CouponSyncTarget,
	scheduler corecron.Scheduler,
	providedLogger *zap.Logger,
	registrar bus.Registrar,
	publishers ...port.IntegrationEventPublisher,
) (*Module, error) {
	return wooruntime.NewWithCouponTarget(cfg, contactService, orderService, couponSyncTarget, scheduler, providedLogger, registrar, publishers...)
}
