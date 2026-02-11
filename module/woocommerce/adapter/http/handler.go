package http

import (
	"context"
	"errors"

	corehttp "mannaiah/module/core/http"
	woocontact "mannaiah/module/woocommerce/application/contact"
)

var (
	// ErrNilService is returned when a nil service dependency is provided.
	ErrNilService = errors.New("woocommerce service must not be nil")
)

// Authorizer defines authentication and authorization behavior required by WooCommerce endpoints.
type Authorizer interface {
	// Require authenticates and authorizes requests using required permissions.
	Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
	// IsUnauthorized reports authentication errors.
	IsUnauthorized(err error) bool
	// IsForbidden reports authorization errors.
	IsForbidden(err error) bool
}

// Handler defines HTTP route handlers for WooCommerce integration endpoints.
type Handler struct {
	// service defines WooCommerce sync service dependencies.
	service woocontact.Service
	// authorizer defines optional auth dependency for protected endpoints.
	authorizer Authorizer
}

// NewHandler creates WooCommerce HTTP handler sets.
func NewHandler(service woocontact.Service, authorizers ...Authorizer) (*Handler, error) {
	if service == nil {
		return nil, ErrNilService
	}

	var authorizer Authorizer
	if len(authorizers) > 0 {
		authorizer = authorizers[0]
	}

	return &Handler{
		service:    service,
		authorizer: authorizer,
	}, nil
}

// SetAuthorizer configures endpoint authentication and authorization dependencies.
func (h *Handler) SetAuthorizer(authorizer Authorizer) {
	if h == nil {
		return
	}

	h.authorizer = authorizer
}

// RegisterRoutes registers WooCommerce integration routes.
func (h *Handler) RegisterRoutes(router corehttp.Router) {
	router.Post("/woo/sync/contacts", h.protect("contacts:manage", h.syncContacts))
}

// syncContacts triggers manual contact sync behavior.
func (h *Handler) syncContacts(ctx corehttp.Context) error {
	summary, err := h.service.SyncContacts(ctx.Context(), "manual")
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
	if errors.Is(err, woocontact.ErrSyncDisabled) {
		return corehttp.NewAppError(503, "woocommerce_contacts_sync_disabled", err)
	}
	if errors.Is(err, woocontact.ErrIntegrationUnavailable) {
		return corehttp.NewAppError(503, "woocommerce_integration_unavailable", err)
	}

	return corehttp.NewAppError(500, "internal_server_error", err)
}
