package http

import (
	"context"
	"errors"

	"mannaiah/module/analytics/application"
	corehttp "mannaiah/module/core/http"
)

var (
	// ErrNilService is returned when nil service dependencies are provided.
	ErrNilService = errors.New("analytics service must not be nil")
)

// Authorizer defines authentication and authorization behavior required by analytics endpoints.
type Authorizer interface {
	// Require authenticates and authorizes requests using required permissions.
	Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
	// IsUnauthorized reports authentication errors.
	IsUnauthorized(err error) bool
	// IsForbidden reports authorization errors.
	IsForbidden(err error) bool
}

// Service defines analytics use-case behavior required by HTTP handlers.
type Service interface {
	// Status returns analytics runtime health values.
	Status(ctx context.Context) application.Status
	// Seed executes best-effort initial data seeding behavior.
	Seed(ctx context.Context) (*application.SeedSummary, error)
}

// Handler defines HTTP route handlers for analytics endpoints.
type Handler struct {
	// service defines analytics use-case dependencies.
	service Service
	// authorizer defines optional auth dependencies.
	authorizer Authorizer
}

// NewHandler creates analytics HTTP handlers.
func NewHandler(service Service, authorizers ...Authorizer) (*Handler, error) {
	if service == nil {
		return nil, ErrNilService
	}

	var authorizer Authorizer
	if len(authorizers) > 0 {
		authorizer = authorizers[0]
	}

	return &Handler{service: service, authorizer: authorizer}, nil
}

// SetAuthorizer configures endpoint authentication and permission dependencies.
func (h *Handler) SetAuthorizer(authorizer Authorizer) {
	if h == nil {
		return
	}

	h.authorizer = authorizer
}

// RegisterRoutes registers analytics routes.
func (h *Handler) RegisterRoutes(router corehttp.Router) {
	router.Get("/analytics/status", h.protect("marketing:manage", h.status))
	router.Post("/analytics/seed", h.protect("marketing:manage", h.seed))
}

// status handles analytics status requests.
func (h *Handler) status(ctx corehttp.Context) error {
	return ctx.Status(200).JSON(h.service.Status(ctx.Context()))
}

// seed handles analytics seed requests.
func (h *Handler) seed(ctx corehttp.Context) error {
	summary, err := h.service.Seed(ctx.Context())
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(summary)
}

// protect wraps endpoint handlers with optional authentication and permission checks.
func (h *Handler) protect(permission string, next corehttp.Handler) corehttp.Handler {
	if h == nil || h.authorizer == nil {
		return next
	}

	return func(ctx corehttp.Context) error {
		err := h.authorizer.Require(ctx.Context(), ctx.GetHeader("Authorization"), permission)
		if err != nil {
			return h.mapError(err)
		}

		return next(ctx)
	}
}

// mapError maps app/auth errors to HTTP-layer app errors.
func (h *Handler) mapError(err error) error {
	if h != nil && h.authorizer != nil {
		if h.authorizer.IsUnauthorized(err) {
			return corehttp.NewAppError(401, "unauthorized", err)
		}
		if h.authorizer.IsForbidden(err) {
			return corehttp.NewAppError(403, "forbidden", err)
		}
	}
	if errors.Is(err, application.ErrDisabled) {
		return corehttp.NewAppError(503, "analytics_disabled", err)
	}

	return corehttp.NewAppError(500, "internal_server_error", err)
}
