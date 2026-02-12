package auth

import (
	"github.com/getkin/kin-openapi/openapi3"
	"go.uber.org/zap"
	authruntime "mannaiah/module/auth/runtime"
	corehttp "mannaiah/module/core/http"
)

// Authorizer defines authentication and authorization behavior required by module adapters.
type Authorizer = authruntime.Authorizer

// Module defines auth-module composition dependencies.
type Module = authruntime.Module

// Loader defines bootstrap hooks required by auth modules.
type Loader interface {
	// RegisterRoutes registers module route handlers.
	RegisterRoutes(register func(router corehttp.Router))
	// AddOpenAPISpec merges module OpenAPI specs.
	AddOpenAPISpec(spec *openapi3.T) error
}

// New creates an auth module with JWT verification and scope authorization support.
func New(cfg Config, coreEnvironment string, logger *zap.Logger) (*Module, error) {
	return authruntime.New(cfg, coreEnvironment, logger)
}

// OpenAPISpec returns auth-module OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	return authruntime.OpenAPISpec()
}
