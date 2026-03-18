package shipping

import (
	"github.com/getkin/kin-openapi/openapi3"
	"go.uber.org/zap"
	shippinghttp "mannaiah/module/shipping/adapter/http"
	shippingruntime "mannaiah/module/shipping/runtime"
)

// Module defines composition-root wiring for shipping endpoints.
type Module = shippingruntime.Module

// Loader defines bootstrap hooks required by shipping modules.
type Loader = shippingruntime.Loader

// Authorizer defines authentication and authorization behavior required by shipping endpoints.
type Authorizer = shippinghttp.Authorizer

// New creates shipping modules with source adapters and route handlers.
func New(cfg Config, providedLogger *zap.Logger) (*Module, error) {
	return shippingruntime.New(cfg, providedLogger)
}

// OpenAPISpec returns shipping-module OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	return shippingruntime.OpenAPISpec()
}
