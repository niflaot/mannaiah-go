package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.uber.org/zap"
	jwtadapter "mannaiah/module/auth/adapter/jwt"
	"mannaiah/module/auth/application"
)

// Authorizer defines authentication and authorization behavior required by module adapters.
type Authorizer interface {
	// Require authenticates and authorizes a request by bearer header and required permissions.
	Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
	// IsUnauthorized reports whether an error is an authentication failure.
	IsUnauthorized(err error) bool
	// IsForbidden reports whether an error is an authorization failure.
	IsForbidden(err error) bool
}

// Module defines auth-module composition dependencies.
type Module struct {
	// service defines application-layer auth behavior.
	service *application.AuthService
}

var (
	// _ ensures Module satisfies authorizer contracts.
	_ Authorizer = (*Module)(nil)
)

// New creates an auth module with JWT verification and scope authorization support.
func New(cfg Config, coreEnvironment string, logger *zap.Logger) (*Module, error) {
	resolvedEnvironment := resolveEnvironment(cfg.NodeEnvironment, coreEnvironment)

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

	return &Module{service: service}, nil
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

// resolveEnvironment resolves runtime environment with optional fallback values.
func resolveEnvironment(nodeEnvironment string, fallback string) string {
	if value := strings.TrimSpace(nodeEnvironment); value != "" {
		return value
	}

	if value := strings.TrimSpace(fallback); value != "" {
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
