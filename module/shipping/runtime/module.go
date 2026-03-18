package runtime

import (
	"github.com/getkin/kin-openapi/openapi3"
	"go.uber.org/zap"
	corehttp "mannaiah/module/core/http"
	shippinghttp "mannaiah/module/shipping/adapter/http"
	quoteservice "mannaiah/module/shipping/application/quote/service"
	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

// Module defines composition-root wiring for shipping endpoints.
type Module struct {
	// cfg defines shipping module configuration values.
	cfg Config
	// service defines shipping quote use-case dependencies.
	service quoteservice.Service
	// handler defines HTTP route adapter dependencies.
	handler *shippinghttp.Handler
	// logger defines structured log dependencies.
	logger *zap.Logger
}

// Loader defines bootstrap hooks required by shipping modules.
type Loader interface {
	// RegisterRoutes registers module route handlers.
	RegisterRoutes(register func(router corehttp.Router))
	// AddOpenAPISpec merges module OpenAPI specs.
	AddOpenAPISpec(spec *openapi3.T) error
}

// New creates shipping modules with source adapters and route handlers.
func New(cfg Config, providedLogger *zap.Logger) (*Module, error) {
	logger := resolveLogger(providedLogger)

	gateway, err := newTCCGateway(cfg, logger)
	if err != nil {
		logger.Warn(
			"shipping integration configuration is invalid; endpoint will remain documented and return 503 until fixed",
			zap.Error(err),
		)
		gateway = failingGateway{err: err}
	}

	service, err := quoteservice.NewService(map[domain.Carrier]port.RateQuoteGateway{
		domain.CarrierTCC: gateway,
	})
	if err != nil {
		return nil, err
	}

	handler, err := shippinghttp.NewHandler(service)
	if err != nil {
		return nil, err
	}

	return &Module{
		cfg:     cfg,
		service: service,
		handler: handler,
		logger:  logger,
	}, nil
}

// RegisterRoutes registers shipping routes on the provided router.
func (m *Module) RegisterRoutes(router corehttp.Router) {
	if m == nil || m.handler == nil {
		return
	}

	m.handler.RegisterRoutes(router)
}

// SetAuthorizer configures endpoint authentication and permission dependencies.
func (m *Module) SetAuthorizer(authorizer shippinghttp.Authorizer) {
	if m == nil || m.handler == nil {
		return
	}

	m.handler.SetAuthorizer(authorizer)
}

// OpenAPISpec returns shipping module OpenAPI documentation.
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
