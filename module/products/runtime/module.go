package runtime

import (
	corehttp "mannaiah/module/core/http"
	producthttp "mannaiah/module/products/adapter/http/product"
	variationhttp "mannaiah/module/products/adapter/http/variation"
	productstore "mannaiah/module/products/adapter/store/product"
	variationstore "mannaiah/module/products/adapter/store/variation"
	productapplication "mannaiah/module/products/application/product"
	variationapplication "mannaiah/module/products/application/variation"

	"github.com/getkin/kin-openapi/openapi3"
	"gorm.io/gorm"
)

// Module defines composition-root wiring for product endpoints.
type Module struct {
	// productHandler defines HTTP adapter used for product route registration.
	productHandler *producthttp.Handler
	// productService defines product application service dependencies.
	productService productapplication.Service
	// variationHandler defines HTTP adapter used for variation route registration.
	variationHandler *variationhttp.Handler
	// variationService defines variation application service dependencies.
	variationService variationapplication.Service
}

// Loader defines bootstrap hooks required by products modules.
type Loader interface {
	// RegisterRoutes registers module route handlers.
	RegisterRoutes(register func(router corehttp.Router))
	// AddOpenAPISpec merges module OpenAPI specs.
	AddOpenAPISpec(spec *openapi3.T) error
}

// New creates a products module with adapter wiring.
func New(db *gorm.DB, assetLookup productapplication.AssetLookup) (*Module, error) {
	productRepository, err := productstore.NewRepository(db)
	if err != nil {
		return nil, err
	}

	productService, err := productapplication.NewService(productRepository, assetLookup)
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

	return &Module{
		productHandler:   productHandler,
		productService:   productService,
		variationHandler: variationHandler,
		variationService: variationService,
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
