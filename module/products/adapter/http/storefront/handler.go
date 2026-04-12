package storefront

import (
	"context"
	"errors"

	corehttp "mannaiah/module/core/http"
	storefrontservice "mannaiah/module/products/application/storefront/service"
)

var (
	// ErrNilService is returned when service dependencies are nil.
	ErrNilService = errors.New("storefront navigation service must not be nil")
)

// Authorizer defines authentication and authorization behavior required by storefront endpoints.
type Authorizer interface {
	// Require authenticates and authorizes requests using required permissions.
	Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
	// IsUnauthorized reports authentication errors.
	IsUnauthorized(err error) bool
	// IsForbidden reports authorization errors.
	IsForbidden(err error) bool
}

// Handler defines HTTP route handlers for storefront navigation endpoints.
type Handler struct {
	// service defines storefront navigation use-case dependencies.
	service storefrontservice.Service
	// authorizer defines optional endpoint auth dependencies.
	authorizer Authorizer
}

// NewHandler creates storefront navigation HTTP handlers.
func NewHandler(service storefrontservice.Service, authorizers ...Authorizer) (*Handler, error) {
	if service == nil {
		return nil, ErrNilService
	}

	var authorizer Authorizer
	if len(authorizers) > 0 {
		authorizer = authorizers[0]
	}

	return &Handler{service: service, authorizer: authorizer}, nil
}

// SetAuthorizer configures auth dependencies for protected endpoints.
func (h *Handler) SetAuthorizer(authorizer Authorizer) {
	if h == nil {
		return
	}

	h.authorizer = authorizer
}

// RegisterRoutes registers storefront navigation endpoints.
func (h *Handler) RegisterRoutes(router corehttp.Router) {
	router.Get("/storefront/navigation", h.protect("storefront:manage", h.navigation))
}

// navigation handles storefront navigation retrieval endpoints.
func (h *Handler) navigation(ctx corehttp.Context) error {
	navigation, err := h.service.Get(ctx.Context())
	if err != nil {
		return corehttp.NewAppError(500, "storefront_navigation_unavailable", err)
	}

	return ctx.Status(200).JSON(navigation)
}

// protect applies optional auth requirements to storefront endpoints.
func (h *Handler) protect(permission string, next corehttp.Handler) corehttp.Handler {
	return func(ctx corehttp.Context) error {
		if h.authorizer == nil {
			return next(ctx)
		}

		err := h.authorizer.Require(ctx.Context(), ctx.GetHeader("Authorization"), permission)
		if err == nil {
			return next(ctx)
		}
		if h.authorizer.IsUnauthorized(err) {
			return corehttp.NewAppError(401, "unauthorized", err)
		}
		if h.authorizer.IsForbidden(err) {
			return corehttp.NewAppError(403, "forbidden", err)
		}

		return corehttp.NewAppError(500, "authorization_failed", err)
	}
}
