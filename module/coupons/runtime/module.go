// Package runtime defines the coupon module composition root.
package runtime

import (
	"errors"

	"github.com/getkin/kin-openapi/openapi3"
	"gorm.io/gorm"
	couponstore "mannaiah/module/coupons/adapter/store"
	couponeventadapter "mannaiah/module/coupons/adapter/event"
	couponhttp "mannaiah/module/coupons/adapter/http"
	couponservice "mannaiah/module/coupons/application/coupon/service"
	corehttp "mannaiah/module/core/http"
	"mannaiah/module/core/messaging/bus"
	"mannaiah/module/coupons/port"
)

var (
	// ErrNilDB is returned when a nil database connection is provided.
	ErrNilDB = errors.New("coupon module: db must not be nil")
)

// Loader defines bootstrap hooks required by the coupon module.
type Loader interface {
	// RegisterRoutes registers module route handlers.
	RegisterRoutes(register func(router corehttp.Router))
	// AddOpenAPISpec merges module OpenAPI specs.
	AddOpenAPISpec(spec *openapi3.T) error
}

// Module defines the coupon module composition root.
type Module struct {
	// service defines coupon use-case dependencies.
	service *couponservice.Service
	// handler defines HTTP route adapter dependencies.
	handler *couponhttp.Handler
}

// New creates the coupon module, wiring all adapters and services.
func New(db *gorm.DB, publisher port.IntegrationEventPublisher) (*Module, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	repo, err := couponstore.NewRepository(db)
	if err != nil {
		return nil, err
	}

	svc, err := couponservice.NewService(repo, repo, publisher)
	if err != nil {
		return nil, err
	}

	handler, err := couponhttp.NewHandler(svc)
	if err != nil {
		return nil, err
	}

	return &Module{service: svc, handler: handler}, nil
}

// NewWithMessaging creates the coupon module wiring a bus publisher adapter.
func NewWithMessaging(db *gorm.DB, busPublisher bus.Publisher) (*Module, error) {
	if busPublisher == nil {
		return New(db, nil)
	}

	pub, err := couponeventadapter.NewPublisher(busPublisher)
	if err != nil {
		return nil, err
	}

	return New(db, pub)
}

// SetAuthorizer configures endpoint authentication dependencies.
func (m *Module) SetAuthorizer(authorizer couponhttp.Authorizer) {
	if m == nil || m.handler == nil {
		return
	}
	m.handler.SetAuthorizer(authorizer)
}

// Service returns the coupon application service.
func (m *Module) Service() *couponservice.Service {
	if m == nil {
		return nil
	}
	return m.service
}

// RegisterRoutes registers coupon routes on the provided router.
func (m *Module) RegisterRoutes(router corehttp.Router) {
	if m == nil || m.handler == nil {
		return
	}
	m.handler.RegisterRoutes(router)
}

// OpenAPISpec returns the coupon module OpenAPI documentation.
func (m *Module) OpenAPISpec() *openapi3.T {
	return OpenAPISpec()
}

// Load mounts all module routes and specs into the provided startup loader.
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
