package runtime

import (
	"context"
	"errors"

	"github.com/getkin/kin-openapi/openapi3"
	"go.uber.org/zap"
	corehttp "mannaiah/module/core/http"
	falabellahttp "mannaiah/module/falabella/adapter/http"
	brandservice "mannaiah/module/falabella/application/brand/service"
	productsyncservice "mannaiah/module/falabella/application/productsync/service"
	"mannaiah/module/falabella/port"
)

var (
	// ErrModuleNotInitialized is returned when module startup methods are called on nil receivers.
	ErrModuleNotInitialized = errors.New("falabella module is not initialized")
)

// Loader defines bootstrap hooks required by Falabella modules.
type Loader interface {
	// RegisterRoutes registers module route handlers.
	RegisterRoutes(register func(router corehttp.Router))
	// AddOpenAPISpec merges module OpenAPI specs.
	AddOpenAPISpec(spec *openapi3.T) error
}

// Module defines composition-root wiring for Falabella endpoints.
type Module struct {
	// cfg defines Falabella integration config values.
	cfg Config
	// service defines Falabella brand service dependencies.
	service brandservice.Service
	// productSyncService defines Falabella product-sync service dependencies.
	productSyncService productsyncservice.Service
	// handler defines HTTP route adapter dependencies.
	handler *falabellahttp.Handler
	// logger defines structured logging dependencies.
	logger *zap.Logger
}

// New creates Falabella modules with source adapters and route handlers.
func New(cfg Config, providedLogger *zap.Logger, catalogs ...port.ProductCatalog) (*Module, error) {
	logger := resolveLogger(providedLogger)

	source, sourceErr := newSource(cfg, logger)
	if sourceErr != nil {
		logger.Warn(
			"falabella integration configuration is invalid; endpoint will remain documented and return 503 until fixed",
			zap.Error(sourceErr),
		)
		source = failingSource{err: sourceErr}
	}

	service, err := brandservice.NewService(source)
	if err != nil {
		return nil, err
	}

	productCatalog := resolveCatalog(catalogs...)
	productSyncService, err := productsyncservice.NewService(source, productCatalog, productsyncservice.Config{
		Realm:            cfg.ProductRealm,
		CategoryID:       cfg.ProductCategoryID,
		GlobalIdentifier: cfg.ProductGlobalIdentifier,
		AttributeSetID:   cfg.ProductAttributeSetID,
	})
	if err != nil {
		return nil, err
	}

	handler, err := falabellahttp.NewHandler(service, productSyncService)
	if err != nil {
		return nil, err
	}

	module := &Module{cfg: cfg, service: service, productSyncService: productSyncService, handler: handler, logger: logger}
	module.validateAtStartup(resolveContext(context.Background()))
	return module, nil
}

// RegisterRoutes registers Falabella routes on the provided router.
func (m *Module) RegisterRoutes(router corehttp.Router) {
	if m == nil || m.handler == nil {
		return
	}

	m.handler.RegisterRoutes(router)
}

// SetAuthorizer configures endpoint authentication and permission dependencies.
func (m *Module) SetAuthorizer(authorizer falabellahttp.Authorizer) {
	if m == nil || m.handler == nil {
		return
	}

	m.handler.SetAuthorizer(authorizer)
}

// OpenAPISpec returns Falabella module OpenAPI documentation.
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

// validateAtStartup verifies integration availability and logs startup warnings.
func (m *Module) validateAtStartup(ctx context.Context) {
	if m == nil || m.service == nil {
		return
	}

	validationCtx, cancel := context.WithTimeout(ctx, resolveValidationTimeout(m.cfg.ValidationTimeoutMS))
	defer cancel()

	if err := m.service.ValidateIntegration(validationCtx); err != nil {
		m.logger.Warn(
			"falabella integration unavailable; endpoints remain documented and return 503 until integration recovers",
			zap.Error(err),
		)
	}
	if err := m.productSyncService.ValidateIntegration(validationCtx); err != nil {
		m.logger.Warn(
			"falabella integration unavailable for product sync; endpoints remain documented and return 503 until integration recovers",
			zap.Error(err),
		)
	}
}
