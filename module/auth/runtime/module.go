package runtime

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"go.uber.org/zap"
	authhttp "mannaiah/module/auth/adapter/http"
	jwtadapter "mannaiah/module/auth/adapter/jwt"
	"mannaiah/module/auth/application"
	corehttp "mannaiah/module/core/http"
)

// Authorizer defines authentication and authorization behavior required by module adapters.
type Authorizer interface {
	// Require authenticates and authorizes a request by bearer header and required permissions.
	Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
	// IsUnauthorized reports whether an error is an authentication failure.
	IsUnauthorized(err error) bool
	// IsForbidden reports whether an error is an authorization failure.
	IsForbidden(err error) bool
	// Subject resolves the caller subject from the authorization header.
	// Returns "system" for dev-bypass tokens or when authentication fails.
	Subject(ctx context.Context, authorizationHeader string) string
}

// Loader defines bootstrap hooks required by auth modules.
type Loader interface {
	// RegisterRoutes registers module route handlers.
	RegisterRoutes(register func(router corehttp.Router))
	// AddOpenAPISpec merges module OpenAPI specs.
	AddOpenAPISpec(spec *openapi3.T) error
}

// Module defines auth-module composition dependencies.
type Module struct {
	// service defines application-layer auth behavior.
	service application.Service
	// handler defines HTTP adapter used for route registration.
	handler *authhttp.Handler
}

var (
	// _ ensures Module satisfies authorizer contracts.
	_ Authorizer = (*Module)(nil)
)

// New creates an auth module with JWT verification and scope authorization support.
func New(cfg Config, coreEnvironment string, logger *zap.Logger) (*Module, error) {
	resolvedEnvironment := resolveEnvironment(coreEnvironment)

	verifier, err := jwtadapter.NewVerifier(jwtadapter.Config{
		Issuer:             strings.TrimSpace(cfg.Issuer),
		Audience:           strings.TrimSpace(cfg.Audience),
		JWKSURL:            buildJWKSURL(cfg.Issuer),
		RateLimitPerMinute: cfg.JWKSRateLimitPerMinute,
		CacheTTL:           time.Duration(resolvePositiveInt(cfg.JWKSCacheTTLMS, 300000)) * time.Millisecond,
		HTTPTimeout:        time.Duration(resolvePositiveInt(cfg.JWKSHTTPTimeoutMS, 5000)) * time.Millisecond,
	})
	if err != nil {
		return nil, err
	}

	service, err := application.NewService(
		resolvedEnvironment,
		strings.TrimSpace(cfg.DevAuthToken),
		strings.TrimSpace(cfg.DevAuthScope),
		verifier,
		logger,
	)
	if err != nil {
		return nil, err
	}

	handler, err := authhttp.NewHandler(service)
	if err != nil {
		return nil, err
	}

	return &Module{service: service, handler: handler}, nil
}

// Require authenticates and authorizes a request by bearer header and required permissions.
func (m *Module) Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error {
	if m == nil || m.service == nil {
		return application.ErrUnauthorized
	}

	return m.service.Require(ctx, authorizationHeader, requiredPermissions...)
}

// IsUnauthorized reports whether an error is an authentication failure.
func (m *Module) IsUnauthorized(err error) bool {
	return errors.Is(err, application.ErrUnauthorized)
}

// IsForbidden reports whether an error is an authorization failure.
func (m *Module) IsForbidden(err error) bool {
	return errors.Is(err, application.ErrForbidden)
}

// Subject resolves the caller subject from the authorization header.
// Returns "system" for dev-bypass tokens or when authentication fails.
func (m *Module) Subject(ctx context.Context, authorizationHeader string) string {
	if m == nil || m.service == nil {
		return "system"
	}
	claims, err := m.service.Authenticate(ctx, authorizationHeader)
	if err != nil {
		return "system"
	}
	if strings.TrimSpace(claims.Subject) == "dev-admin" {
		return "system"
	}

	return strings.TrimSpace(claims.Subject)
}

// RegisterRoutes registers auth routes on the provided router.
func (m *Module) RegisterRoutes(router corehttp.Router) {
	if m == nil || m.handler == nil {
		return
	}

	m.handler.RegisterRoutes(router)
}

// OpenAPISpec returns auth-module OpenAPI documentation.
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

// resolveEnvironment resolves runtime environment from core-level configuration.
func resolveEnvironment(coreEnvironment string) string {
	if value := strings.TrimSpace(coreEnvironment); value != "" {
		return value
	}

	return "development"
}

// buildJWKSURL resolves the default JWKS URL from the configured issuer.
func buildJWKSURL(issuer string) string {
	trimmed := strings.TrimSpace(issuer)
	trimmed = strings.TrimRight(trimmed, "/")
	if trimmed == "" {
		return ""
	}

	return trimmed + "/jwks"
}

// resolvePositiveInt returns value when positive, otherwise fallback.
func resolvePositiveInt(value int, fallback int) int {
	if value > 0 {
		return value
	}

	return fallback
}
