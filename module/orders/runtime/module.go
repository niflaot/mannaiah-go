package runtime

import (
	"context"
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	"gorm.io/gorm"
	corehttp "mannaiah/module/core/http"
	ordershttp "mannaiah/module/orders/adapter/http"
	ordersstore "mannaiah/module/orders/adapter/store"
	ordersapplication "mannaiah/module/orders/application"
	ordersport "mannaiah/module/orders/port"
)

// Module defines composition-root wiring for order endpoints.
type Module struct {
	// handler defines HTTP adapter used for route registration.
	handler *ordershttp.Handler
	// service defines order application service dependencies.
	service ordersapplication.Service
}

// Loader defines bootstrap hooks required by order modules.
type Loader interface {
	// RegisterRoutes registers module route handlers.
	RegisterRoutes(register func(router corehttp.Router))
	// AddOpenAPISpec merges module OpenAPI specs.
	AddOpenAPISpec(spec *openapi3.T) error
}

// New creates an orders module with schema migration and adapter wiring.
func New(db *gorm.DB, customerSource ordersport.CustomerSource, resolvers ...ordersport.ProductResolver) (*Module, error) {
	repository, err := ordersstore.NewRepository(db)
	if err != nil {
		return nil, err
	}
	if err := repository.EnsureSchema(context.Background()); err != nil {
		return nil, fmt.Errorf("ensure orders schema: %w", err)
	}

	service, err := ordersapplication.NewService(repository, customerSource, resolvers...)
	if err != nil {
		return nil, err
	}

	handler, err := ordershttp.NewHandler(service)
	if err != nil {
		return nil, err
	}

	return &Module{handler: handler, service: service}, nil
}

// RegisterRoutes registers order routes on the provided router.
func (m *Module) RegisterRoutes(router corehttp.Router) {
	if m == nil || m.handler == nil {
		return
	}

	m.handler.RegisterRoutes(router)
}

// Service returns order application service dependencies for module integrations.
func (m *Module) Service() ordersapplication.Service {
	if m == nil {
		return nil
	}

	return m.service
}

// SetAuthorizer configures endpoint authentication and permission dependencies.
func (m *Module) SetAuthorizer(authorizer ordershttp.Authorizer) {
	if m == nil || m.handler == nil {
		return
	}

	m.handler.SetAuthorizer(authorizer)
}

// OpenAPISpec returns order-module OpenAPI documentation.
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
