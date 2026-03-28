package http

import (
	"context"
	"errors"

	"mannaiah/module/auth/application"
	"mannaiah/module/auth/domain"
	corehttp "mannaiah/module/core/http"
)

var (
	// ErrNilService is returned when service dependencies are nil.
	ErrNilService = errors.New("auth service must not be nil")
)

// Service defines auth behavior required by HTTP endpoints.
type Service interface {
	// Require authenticates request headers and validates required permissions.
	Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
	// Authenticate validates request headers and returns principal claims.
	Authenticate(ctx context.Context, authorizationHeader string) (*domain.Claims, error)
}

// Handler defines HTTP route handlers for auth endpoints.
type Handler struct {
	// service defines auth use-case dependencies.
	service Service
}

// checkAuthResponse defines check-auth response payload values.
type checkAuthResponse struct {
	// Status defines authentication status values.
	Status string `json:"status"`
}

// malformationIssue defines one detected permission dependency violation.
type malformationIssue struct {
	// Permission is the scope held by the user that triggered this check.
	Permission string `json:"permission"`
	// Requires is the missing dependency permission.
	Requires string `json:"requires"`
	// Description explains the dependency in human-readable form.
	Description string `json:"description"`
}

// malformationResponse defines the /users/malformation response payload.
type malformationResponse struct {
	// Status is "ok" when no issues are found, "malformed" otherwise.
	Status string `json:"status"`
	// Issues lists each detected dependency violation. Empty when status is "ok".
	Issues []malformationIssue `json:"issues"`
}

// NewHandler creates auth HTTP handlers.
func NewHandler(service Service) (*Handler, error) {
	if service == nil {
		return nil, ErrNilService
	}

	return &Handler{service: service}, nil
}

// RegisterRoutes registers auth endpoints.
func (h *Handler) RegisterRoutes(router corehttp.Router) {
	router.Get("/check-auth", h.checkAuth)
	router.Get("/users/malformation", h.checkMalformation)
}

// checkAuth validates JWT authentication and returns authenticated status when valid.
func (h *Handler) checkAuth(ctx corehttp.Context) error {
	err := h.service.Require(ctx.Context(), ctx.GetHeader("Authorization"))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(checkAuthResponse{Status: "authenticated"})
}

// checkMalformation authenticates the request and checks for cross-domain permission
// dependency violations. Returns "ok" when the token's scope set is coherent, or
// "malformed" with a list of missing dependencies otherwise.
func (h *Handler) checkMalformation(ctx corehttp.Context) error {
	claims, err := h.service.Authenticate(ctx.Context(), ctx.GetHeader("Authorization"))
	if err != nil {
		return h.mapError(err)
	}

	scopes := claims.Scopes()
	issues := detectMalformations(scopes)

	status := "ok"
	if len(issues) > 0 {
		status = "malformed"
	}

	return ctx.Status(200).JSON(malformationResponse{
		Status: status,
		Issues: issues,
	})
}

// detectMalformations evaluates all dependency rules against the provided scopes
// and returns a list of violations.
func detectMalformations(scopes []string) []malformationIssue {
	var issues []malformationIssue

	for _, rule := range domain.DependencyRules {
		if !hasScopeOrCover(scopes, rule.Subject) {
			continue
		}

		for _, required := range rule.Required {
			if !hasScopeOrCover(scopes, required) {
				issues = append(issues, malformationIssue{
					Permission:  rule.Subject,
					Requires:    required,
					Description: rule.Description,
				})
			}
		}
	}

	return issues
}

// hasScopeOrCover reports whether any scope in scopes satisfies required,
// applying the same manage wildcard and intermediate hierarchy as the auth service.
func hasScopeOrCover(scopes []string, required string) bool {
	for _, s := range scopes {
		if s == required {
			return true
		}
	}

	// manage wildcard: resource:manage satisfies resource:*
	idx := indexColon(required)
	if idx > 0 {
		manage := required[:idx] + ":manage"
		for _, s := range scopes {
			if s == manage {
				return true
			}
		}
	}

	// intermediate hierarchy from domain.PermissionCovers
	for intermediate, covered := range domain.PermissionCovers {
		for _, c := range covered {
			if c != required {
				continue
			}
			for _, s := range scopes {
				if s == intermediate {
					return true
				}
			}
		}
	}

	return false
}

// indexColon returns the index of ':' in s, or -1 if not found.
func indexColon(s string) int {
	for i, ch := range s {
		if ch == ':' {
			return i
		}
	}

	return -1
}

// mapError maps auth errors to HTTP-layer app errors.
func (h *Handler) mapError(err error) error {
	if errors.Is(err, application.ErrUnauthorized) || errors.Is(err, application.ErrForbidden) {
		return corehttp.NewAppError(401, "unauthorized", err)
	}

	return corehttp.NewAppError(500, "internal_server_error", err)
}
