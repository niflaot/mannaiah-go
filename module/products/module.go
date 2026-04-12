package products

import (
	"github.com/getkin/kin-openapi/openapi3"
	"gorm.io/gorm"
	corecache "mannaiah/module/core/cache"
	corehttp "mannaiah/module/core/http"
	productapplication "mannaiah/module/products/application/product"
	productsruntime "mannaiah/module/products/runtime"

	"go.uber.org/zap"
)

// Module defines composition-root wiring for product endpoints.
type Module = productsruntime.Module

// Config defines product runtime feature configuration.
type Config = productsruntime.Config

// Loader defines bootstrap hooks required by products modules.
type Loader interface {
	// RegisterRoutes registers module route handlers.
	RegisterRoutes(register func(router corehttp.Router))
	// AddOpenAPISpec merges module OpenAPI specs.
	AddOpenAPISpec(spec *openapi3.T) error
}

// New creates a products module with schema migration and adapter wiring.
func New(db *gorm.DB, assetLookup productapplication.AssetLookup) (*Module, error) {
	return productsruntime.New(db, assetLookup)
}

// NewWithConfig creates a products module with runtime configuration and optional cache/logging dependencies.
func NewWithConfig(
	db *gorm.DB,
	assetLookup productapplication.AssetLookup,
	cfg Config,
	cacheStore corecache.Store,
	logger *zap.Logger,
) (*Module, error) {
	return productsruntime.NewWithConfig(db, assetLookup, cfg, cacheStore, logger)
}

// OpenAPISpec returns product-module OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	return productsruntime.OpenAPISpec()
}
