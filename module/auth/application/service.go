package application

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"mannaiah/module/auth/domain"
	"mannaiah/module/auth/port"
)

// Service defines auth use cases for request authentication and permission checks.
type Service interface {
	// Require authenticates request headers and validates required permissions.
	Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
	// Authenticate validates request headers and returns principal claims.
	Authenticate(ctx context.Context, authorizationHeader string) (*domain.Claims, error)
	// Authorize validates principal scopes against required permissions.
	Authorize(claims *domain.Claims, requiredPermissions ...string) error
}

// AuthService defines auth application use-case dependencies.
type AuthService struct {
	// environment defines runtime environment used for dev bypass checks.
	environment string
	// devAuthToken defines optional development bypass bearer token.
	devAuthToken string
	// devAuthScope defines optional scopes assigned to bypass principal.
	devAuthScope string
	// verifier defines JWT verification dependency.
	verifier port.TokenVerifier
	// logger defines structured logging dependency.
	logger *zap.Logger
}

var (
	// _ ensures AuthService satisfies service contracts.
	_ Service = (*AuthService)(nil)
)

// NewService creates auth use-case services with verifier and logging dependencies.
func NewService(environment string, devAuthToken string, devAuthScope string, verifier port.TokenVerifier, logger *zap.Logger) (*AuthService, error) {
	if verifier == nil {
		return nil, ErrNilVerifier
	}

	resolvedLogger := logger
	if resolvedLogger == nil {
		resolvedLogger = zap.NewNop()
	}

	return &AuthService{
		environment:  strings.TrimSpace(environment),
		devAuthToken: strings.TrimSpace(devAuthToken),
		devAuthScope: strings.TrimSpace(devAuthScope),
		verifier:     verifier,
		logger:       resolvedLogger,
	}, nil
}

// Require authenticates request headers and validates required permissions.
func (s *AuthService) Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error {
	claims, err := s.Authenticate(ctx, authorizationHeader)
	if err != nil {
		return err
	}

	if err := s.Authorize(claims, requiredPermissions...); err != nil {
		return err
	}

	return nil
}

// Authenticate validates request headers and returns principal claims.
func (s *AuthService) Authenticate(ctx context.Context, authorizationHeader string) (*domain.Claims, error) {
	if s.matchesDevBypass(authorizationHeader) {
		s.logger.Debug("Using Dev Auth Token Bypass")
		return &domain.Claims{
			Subject: "dev-admin",
			Scope:   s.devAuthScope,
			Raw: map[string]any{
				"roles": []string{"admin"},
			},
		}, nil
	}

	token, err := parseBearerToken(authorizationHeader)
	if err != nil {
		return nil, err
	}

	claims, err := s.verifier.Verify(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnauthorized, err)
	}

	return claims, nil
}

// Authorize validates principal scopes against required permissions.
func (s *AuthService) Authorize(claims *domain.Claims, requiredPermissions ...string) error {
	if len(requiredPermissions) == 0 {
		return nil
	}
	if claims == nil {
		return ErrForbidden
	}

	scopes := claims.Scopes()
	if len(scopes) == 0 {
		return ErrForbidden
	}

	for _, permission := range requiredPermissions {
		required := strings.TrimSpace(permission)
		if required == "" {
			continue
		}
		if !hasPermission(scopes, required) {
			return fmt.Errorf("%w: %s", ErrForbidden, required)
		}
	}

	return nil
}

// matchesDevBypass reports whether dev bypass should authenticate the request.
func (s *AuthService) matchesDevBypass(authorizationHeader string) bool {
	if strings.TrimSpace(s.environment) != "development" {
		return false
	}
	if strings.TrimSpace(s.devAuthToken) == "" {
		return false
	}

	expected := "Bearer " + s.devAuthToken
	return authorizationHeader == expected
}

// parseBearerToken parses Bearer authorization headers into JWT token values.
func parseBearerToken(authorizationHeader string) (string, error) {
	parts := strings.SplitN(strings.TrimSpace(authorizationHeader), " ", 2)
	if len(parts) != 2 {
		return "", ErrUnauthorized
	}
	if parts[0] != "Bearer" {
		return "", ErrUnauthorized
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", ErrUnauthorized
	}

	return token, nil
}

// hasPermission reports whether any scope satisfies required permission values.
// It applies:
//  1. Exact match.
//  2. Manage wildcard: resource:manage satisfies any resource:action.
//  3. Intermediate hierarchy: permissions listed in domain.PermissionCovers.
func hasPermission(scopes []string, required string) bool {
	for _, scope := range scopes {
		if scope == required {
			return true
		}
	}

	// Manage wildcard: resource:manage covers any resource:action.
	manage := managePermission(required)
	if manage != "" {
		for _, scope := range scopes {
			if scope == manage {
				return true
			}
		}
	}

	// Intermediate hierarchy: a scope may cover the required permission
	// without being a full manage wildcard (e.g. product:edit covers product:view).
	for intermediate, covered := range domain.PermissionCovers {
		for _, c := range covered {
			if c != required {
				continue
			}
			for _, scope := range scopes {
				if scope == intermediate {
					return true
				}
			}
		}
	}

	return false
}

// managePermission resolves resource-level manage permission from required actions.
func managePermission(required string) string {
	index := strings.Index(required, ":")
	if index <= 0 {
		return ""
	}

	return required[:index] + ":manage"
}
