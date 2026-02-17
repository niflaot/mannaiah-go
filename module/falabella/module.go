package falabella

import (
	"github.com/getkin/kin-openapi/openapi3"
	"go.uber.org/zap"
	falabellahttp "mannaiah/module/falabella/adapter/http"
	falabellaruntime "mannaiah/module/falabella/runtime"
	"mannaiah/module/falabella/port"
)

var (
	// ErrModuleNotInitialized is returned when module startup methods are called on nil receivers.
	ErrModuleNotInitialized = falabellaruntime.ErrModuleNotInitialized
)

// Module defines composition-root wiring for Falabella endpoints.
type Module = falabellaruntime.Module

// Loader defines bootstrap hooks required by Falabella modules.
type Loader = falabellaruntime.Loader

// Authorizer defines authentication and authorization behavior required by Falabella endpoints.
type Authorizer = falabellahttp.Authorizer

// ProductCatalog defines cross-module product lookup behavior used by Falabella sync endpoints.
type ProductCatalog = port.ProductCatalog

// New creates Falabella modules with source adapters and route handlers.
func New(cfg Config, providedLogger *zap.Logger, catalogs ...port.ProductCatalog) (*Module, error) {
	return falabellaruntime.New(cfg, providedLogger, catalogs...)
}

// OpenAPISpec returns Falabella module OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	return falabellaruntime.OpenAPISpec()
}
