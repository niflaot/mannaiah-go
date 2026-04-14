package runtime

import (
	corecache "mannaiah/module/core/cache"
	corecron "mannaiah/module/core/cron"
	corehttp "mannaiah/module/core/http"
	categoryhttp "mannaiah/module/products/adapter/http/category"
	producthttp "mannaiah/module/products/adapter/http/product"
	storefronthttp "mannaiah/module/products/adapter/http/storefront"
	taghttp "mannaiah/module/products/adapter/http/tag"
	variationhttp "mannaiah/module/products/adapter/http/variation"
	categorystore "mannaiah/module/products/adapter/store/category"
	productstore "mannaiah/module/products/adapter/store/product"
	tagstore "mannaiah/module/products/adapter/store/tag"
	variationstore "mannaiah/module/products/adapter/store/variation"
	categoryapplication "mannaiah/module/products/application/category"
	productapplication "mannaiah/module/products/application/product"
	storefrontservice "mannaiah/module/products/application/storefront/service"
	tagapplication "mannaiah/module/products/application/tag"
	variationapplication "mannaiah/module/products/application/variation"
	"strings"
	"sync"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Module defines composition-root wiring for product endpoints.
type Module struct {
	// cfg defines runtime configuration values.
	cfg Config
	// storefrontSource defines navigation source dependencies that can be enriched post-construction.
	storefrontSource *storefrontNavigationSource
	// productHandler defines HTTP adapter used for product route registration.
	productHandler *producthttp.Handler
	// productService defines product application service dependencies.
	productService productapplication.Service
	// variationHandler defines HTTP adapter used for variation route registration.
	variationHandler *variationhttp.Handler
	// variationService defines variation application service dependencies.
	variationService variationapplication.Service
	// categoryHandler defines HTTP adapter used for category route registration.
	categoryHandler *categoryhttp.Handler
	// categoryService defines category application service dependencies.
	categoryService categoryapplication.Service
	// storefrontHandler defines HTTP adapter used for storefront route registration.
	storefrontHandler *storefronthttp.Handler
	// storefrontService defines storefront navigation use-case dependencies.
	storefrontService storefrontservice.Service
	// tagHandler defines HTTP adapter used for tag route registration.
	tagHandler *taghttp.Handler
	// tagService defines tag application service dependencies.
	tagService tagapplication.Service
	// scheduler defines optional cron scheduler dependencies.
	scheduler corecron.Scheduler
	// storefrontRefreshEntryID defines optional scheduled storefront-refresh entry identifiers.
	storefrontRefreshEntryID corecron.EntryID
	// logger defines structured logging dependencies.
	logger *zap.Logger
	// mutex guards scheduler lifecycle state.
	mutex sync.Mutex
	// started reports whether scheduler lifecycle start logic has completed.
	started bool
}

// Loader defines bootstrap hooks required by products modules.
type Loader interface {
	// RegisterRoutes registers module route handlers.
	RegisterRoutes(register func(router corehttp.Router))
	// AddOpenAPISpec merges module OpenAPI specs.
	AddOpenAPISpec(spec *openapi3.T) error
}

// New creates a products module with default runtime configuration.
func New(db *gorm.DB, assetLookup productapplication.AssetLookup) (*Module, error) {
	return NewWithConfig(db, assetLookup, Config{}, nil, nil)
}

// NewWithConfig creates a products module with adapter wiring and runtime configuration.
func NewWithConfig(
	db *gorm.DB,
	assetLookup productapplication.AssetLookup,
	cfg Config,
	cacheStore corecache.Store,
	providedLogger *zap.Logger,
) (*Module, error) {
	resolvedConfig := normalizeRuntimeConfig(cfg)

	tagRepository, err := tagstore.NewRepository(db)
	if err != nil {
		return nil, err
	}

	tagService, err := tagapplication.NewService(tagRepository)
	if err != nil {
		return nil, err
	}

	tagHandler, err := taghttp.NewHandler(tagService)
	if err != nil {
		return nil, err
	}

	productRepository, err := productstore.NewRepository(db)
	if err != nil {
		return nil, err
	}

	productService, err := productapplication.NewService(productRepository, assetLookup, tagService)
	if err != nil {
		return nil, err
	}

	productHandler, err := producthttp.NewHandler(productService)
	if err != nil {
		return nil, err
	}

	variationRepository, err := variationstore.NewRepository(db)
	if err != nil {
		return nil, err
	}

	variationService, err := variationapplication.NewService(variationRepository)
	if err != nil {
		return nil, err
	}

	variationHandler, err := variationhttp.NewHandler(variationService)
	if err != nil {
		return nil, err
	}

	categoryRepository, err := categorystore.NewRepository(db)
	if err != nil {
		return nil, err
	}

	categoryService, err := categoryapplication.NewService(categoryRepository)
	if err != nil {
		return nil, err
	}

	categoryHandler, err := categoryhttp.NewHandler(categoryService)
	if err != nil {
		return nil, err
	}

	navigationSource := &storefrontNavigationSource{categoryService: categoryService}
	storefrontNavigationService, err := storefrontservice.NewService(
		navigationSource,
		cacheStore,
		storefrontServiceConfig(resolvedConfig),
		providedLogger,
	)
	if err != nil {
		return nil, err
	}
	productService.SetStorefrontNavigationRefresher(storefrontNavigationService)
	categoryService.SetStorefrontNavigationRefresher(storefrontNavigationService)

	storefrontHandler, err := storefronthttp.NewHandler(storefrontNavigationService)
	if err != nil {
		return nil, err
	}

	return &Module{
		cfg:               resolvedConfig,
		storefrontSource:  navigationSource,
		productHandler:    productHandler,
		productService:    productService,
		variationHandler:  variationHandler,
		variationService:  variationService,
		categoryHandler:   categoryHandler,
		categoryService:   categoryService,
		storefrontHandler: storefrontHandler,
		storefrontService: storefrontNavigationService,
		tagHandler:        tagHandler,
		tagService:        tagService,
		logger:            resolveRuntimeLogger(providedLogger),
	}, nil
}

// RegisterRoutes registers product routes on the provided router.
func (m *Module) RegisterRoutes(router corehttp.Router) {
	if m == nil {
		return
	}

	if m.productHandler != nil {
		m.productHandler.RegisterRoutes(router)
	}
	if m.variationHandler != nil {
		m.variationHandler.RegisterRoutes(router)
	}
	if m.categoryHandler != nil {
		m.categoryHandler.RegisterRoutes(router)
	}
	if m.storefrontHandler != nil {
		m.storefrontHandler.RegisterRoutes(router)
	}
	if m.tagHandler != nil {
		m.tagHandler.RegisterRoutes(router)
	}
}

// Service returns product application service dependencies for module integrations.
func (m *Module) Service() productapplication.Service {
	if m == nil {
		return nil
	}

	return m.productService
}

// VariationService returns variation application service dependencies for module integrations.
func (m *Module) VariationService() variationapplication.Service {
	if m == nil {
		return nil
	}

	return m.variationService
}

// SetAuthorizer configures endpoint authentication and permission dependencies.
func (m *Module) SetAuthorizer(authorizer producthttp.Authorizer) {
	if m == nil {
		return
	}

	if m.productHandler != nil {
		m.productHandler.SetAuthorizer(authorizer)
	}
	if m.variationHandler != nil {
		m.variationHandler.SetAuthorizer(authorizer)
	}
	if m.categoryHandler != nil {
		m.categoryHandler.SetAuthorizer(authorizer)
	}
	if m.storefrontHandler != nil {
		m.storefrontHandler.SetAuthorizer(authorizer)
	}
	if m.tagHandler != nil {
		m.tagHandler.SetAuthorizer(authorizer)
	}
}

// SetStorefrontStaticPageSource configures optional static-page lookups for navigation snapshots.
func (m *Module) SetStorefrontStaticPageSource(source StorefrontStaticPageSource) {
	if m == nil || m.storefrontSource == nil {
		return
	}

	m.storefrontSource.pageSource = source
}

// CategoryService returns category application service dependencies for module integrations.
func (m *Module) CategoryService() categoryapplication.Service {
	if m == nil {
		return nil
	}

	return m.categoryService
}

// TagService returns tag application service dependencies for module integrations.
func (m *Module) TagService() tagapplication.Service {
	if m == nil {
		return nil
	}

	return m.tagService
}

// StorefrontService returns storefront navigation use-case dependencies for module integrations.
func (m *Module) StorefrontService() storefrontservice.Service {
	if m == nil {
		return nil
	}

	return m.storefrontService
}

// OpenAPISpec returns product-module OpenAPI documentation.
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

// normalizeRuntimeConfig resolves product runtime configuration defaults.
func normalizeRuntimeConfig(cfg Config) Config {
	if stringsTrimmed(cfg.StorefrontNavigationRealm) == "" {
		cfg.StorefrontNavigationRealm = "default"
	}
	if cfg.StorefrontNavigationRefreshHours <= 0 {
		cfg.StorefrontNavigationRefreshHours = 12
	}
	if cfg.StorefrontNavigationCacheMultiplier <= 0 {
		cfg.StorefrontNavigationCacheMultiplier = 2
	}
	if cfg.StorefrontNavigationFailureExtensionHours <= 0 {
		cfg.StorefrontNavigationFailureExtensionHours = cfg.StorefrontNavigationRefreshHours
	}
	if stringsTrimmed(cfg.StorefrontNavigationCacheKey) == "" {
		cfg.StorefrontNavigationCacheKey = "products:storefront:navigation:" + cfg.StorefrontNavigationRealm
	}
	if cfg.StorefrontNavigationRegenerationTimeoutSeconds <= 0 {
		cfg.StorefrontNavigationRegenerationTimeoutSeconds = 30
	}

	return cfg
}

// storefrontServiceConfig resolves storefront navigation service configuration from runtime values.
func storefrontServiceConfig(cfg Config) storefrontservice.Config {
	refreshInterval := time.Duration(cfg.StorefrontNavigationRefreshHours) * time.Hour

	return storefrontservice.Config{
		Enabled:             cfg.StorefrontNavigationEnabled,
		Realm:               cfg.StorefrontNavigationRealm,
		CacheKey:            cfg.StorefrontNavigationCacheKey,
		RefreshInterval:     refreshInterval,
		CacheTTL:            refreshInterval * time.Duration(cfg.StorefrontNavigationCacheMultiplier),
		FailureExtensionTTL: time.Duration(cfg.StorefrontNavigationFailureExtensionHours) * time.Hour,
		CollectionBasePath:  "/collections",
		ProductBasePath:     "/product",
		RegenerationTimeout: time.Duration(cfg.StorefrontNavigationRegenerationTimeoutSeconds) * time.Second,
	}
}

// storefrontRefreshSpec resolves the storefront navigation cron descriptor.
func (m *Module) storefrontRefreshSpec() string {
	return "@every " + (time.Duration(m.cfg.StorefrontNavigationRefreshHours) * time.Hour).String()
}

// storefrontRegenerationTimeout resolves the storefront navigation regeneration timeout.
func (m *Module) storefrontRegenerationTimeout() time.Duration {
	return time.Duration(m.cfg.StorefrontNavigationRegenerationTimeoutSeconds) * time.Second
}

// resolveRuntimeLogger resolves nil loggers to no-op defaults.
func resolveRuntimeLogger(providedLogger *zap.Logger) *zap.Logger {
	if providedLogger != nil {
		return providedLogger
	}

	return zap.NewNop()
}

// stringsTrimmed resolves whitespace-trimmed string values.
func stringsTrimmed(value string) string {
	return strings.TrimSpace(value)
}
