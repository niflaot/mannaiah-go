package shopify

import (
	contactapplication "mannaiah/module/contacts/application"
	corehttp "mannaiah/module/core/http"
	"mannaiah/module/core/messaging/bus"
	ordersapplication "mannaiah/module/orders/application"
	shopifyport "mannaiah/module/shopify/port"
	shopifyruntime "mannaiah/module/shopify/runtime"

	"github.com/getkin/kin-openapi/openapi3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	// ErrNilDB is returned when DB dependencies are nil.
	ErrNilDB = shopifyruntime.ErrNilDB
	// ErrNilContactService is returned when contact service dependencies are nil.
	ErrNilContactService = shopifyruntime.ErrNilContactService
	// ErrNilOrderService is returned when order service dependencies are nil.
	ErrNilOrderService = shopifyruntime.ErrNilOrderService
	// ErrModuleNotInitialized is returned when module lifecycle methods are called on nil receivers.
	ErrModuleNotInitialized = shopifyruntime.ErrModuleNotInitialized
)

// Module defines composition-root wiring for Shopify endpoints and consumers.
type Module = shopifyruntime.Module

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
	return shopifyruntime.New(cfg, db, contactService, orderService, providedLogger, registrar, publishers...)
}
