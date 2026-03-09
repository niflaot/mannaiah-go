package runtime

import (
	"context"
	"errors"
	"sync"

	corecron "mannaiah/module/core/cron"
	corehttp "mannaiah/module/core/http"
	falabellahttp "mannaiah/module/falabella/adapter/http"
	"mannaiah/module/falabella/adapter/store"
	brandservice "mannaiah/module/falabella/application/brand/service"
	productsyncservice "mannaiah/module/falabella/application/productsync/service"
	syncstatusservice "mannaiah/module/falabella/application/syncstatus/service"
	"mannaiah/module/falabella/port"

	"github.com/getkin/kin-openapi/openapi3"
	"go.uber.org/zap"
	"gorm.io/gorm"
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
	// syncStatusService defines Falabella sync status service dependencies.
	syncStatusService syncstatusservice.Service
	// handler defines HTTP route adapter dependencies.
	handler *falabellahttp.Handler
	// logger defines structured logging dependencies.
	logger *zap.Logger
	// scheduler defines optional cron scheduler dependencies.
	scheduler corecron.Scheduler
	// schedulerEntryID defines the cron entry identifier for feed status resolution.
	schedulerEntryID corecron.EntryID
	// mutex guards scheduler lifecycle state.
	mutex sync.Mutex
	// started reports whether scheduler lifecycle start logic has completed.
	started bool
}

// Option defines functional option values for Falabella module construction.
type Option func(*moduleOptions)

// moduleOptions defines optional dependencies for module construction.
type moduleOptions struct {
	// db defines optional GORM database dependencies for sync status persistence.
	db *gorm.DB
	// catalogs defines optional product-catalog dependencies.
	catalogs []port.ProductCatalog
}

// WithDB configures database dependencies for sync status persistence.
func WithDB(db *gorm.DB) Option {
	return func(opts *moduleOptions) {
		opts.db = db
	}
}

// WithCatalog configures product-catalog dependencies.
func WithCatalog(catalog port.ProductCatalog) Option {
	return func(opts *moduleOptions) {
		opts.catalogs = append(opts.catalogs, catalog)
	}
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
		Realm:                 cfg.ProductRealm,
		CategoryID:            cfg.ProductCategoryID,
		GlobalIdentifier:      cfg.ProductGlobalIdentifier,
		AttributeSetID:        cfg.ProductAttributeSetID,
		OperatorCode:          cfg.ProductOperatorCode,
		SyncWorkers:           cfg.ProductSyncWorkers,
		ImageTranscodeEnabled: cfg.ProductImageTranscodeEnabled,
		ImageTranscodeBaseURL: cfg.ProductImageTranscodePublicBaseURL,
	})
	if err != nil {
		return nil, err
	}

	handler, err := falabellahttp.NewHandler(service, productSyncService)
	if err != nil {
		return nil, err
	}
	handler.SetImageTranscodeConfig(resolveImageTranscodeConfig(cfg))

	module := &Module{cfg: cfg, service: service, productSyncService: productSyncService, handler: handler, logger: logger}
	productSyncService.SetLogger(logger)
	module.validateAtStartup(resolveContext(context.Background()))
	return module, nil
}

// ConfigureSyncStatus wires sync status persistence and endpoints backed by the provided database.
func (m *Module) ConfigureSyncStatus(db *gorm.DB) error {
	if m == nil || m.handler == nil {
		return ErrModuleNotInitialized
	}
	if db == nil {
		return nil
	}

	repo, err := store.NewRepository(db)
	if err != nil {
		m.logger.Warn("falabella sync status repository initialization failed", zap.Error(err))
		return nil
	}

	// Build source for feed status - reuse the same source used by the module.
	source, sourceErr := newSource(m.cfg, m.logger)
	if sourceErr != nil {
		source = failingSource{err: sourceErr}
	}

	svc, svcErr := syncstatusservice.NewService(repo, source)
	if svcErr != nil {
		m.logger.Warn("falabella sync status service initialization failed", zap.Error(svcErr))
		return nil
	}

	m.syncStatusService = svc

	// Wire sync status recording into product sync flow.
	if productSyncSvc, ok := m.productSyncService.(*productsyncservice.ProductSyncService); ok {
		productSyncSvc.SetRecorder(svc)
	}

	// Rebuild handler with sync status service
	handler, handlerErr := falabellahttp.NewHandler(m.service, m.productSyncService, svc)
	if handlerErr != nil {
		return handlerErr
	}
	handler.SetImageTranscodeConfig(resolveImageTranscodeConfig(m.cfg))
	m.handler = handler

	return nil
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
}
