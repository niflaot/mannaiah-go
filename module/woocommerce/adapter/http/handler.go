package http

import (
	"context"
	"errors"
	"strconv"
	"strings"

	corehttp "mannaiah/module/core/http"
	woocontactservice "mannaiah/module/woocommerce/application/contact/service"
	woocouponservice "mannaiah/module/woocommerce/application/coupon/service"
	wooorderservice "mannaiah/module/woocommerce/application/order/service"
)

var (
	// ErrNilContactService is returned when a nil contact service dependency is provided.
	ErrNilContactService = errors.New("woocommerce contacts service must not be nil")
	// ErrNilOrderService is returned when a nil order service dependency is provided.
	ErrNilOrderService = errors.New("woocommerce orders service must not be nil")
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
	// contactsService defines WooCommerce contact sync service dependencies.
	contactsService woocontactservice.Service
	// ordersService defines WooCommerce order sync service dependencies.
	ordersService wooorderservice.Service
	// couponsService defines WooCommerce coupon sync service dependencies.
	couponsService woocouponservice.Service
	// authorizer defines optional auth dependency for protected endpoints.
	authorizer Authorizer
}

// NewHandler creates WooCommerce HTTP handler sets.
func NewHandler(contactsService woocontactservice.Service, ordersService wooorderservice.Service, authorizers ...Authorizer) (*Handler, error) {
	if contactsService == nil {
		return nil, ErrNilContactService
	}
	if ordersService == nil {
		return nil, ErrNilOrderService
	}

	var authorizer Authorizer
	if len(authorizers) > 0 {
		authorizer = authorizers[0]
	}

	return &Handler{
		contactsService: contactsService,
		ordersService:   ordersService,
		authorizer:      authorizer,
	}, nil
}

// SetAuthorizer configures endpoint authentication and authorization dependencies.
func (h *Handler) SetAuthorizer(authorizer Authorizer) {
	if h == nil {
		return
	}

	h.authorizer = authorizer
}

// SetCouponSyncService configures optional coupon sync dependencies.
func (h *Handler) SetCouponSyncService(service woocouponservice.Service) {
	if h == nil {
		return
	}

	h.couponsService = service
}

// RegisterRoutes registers WooCommerce integration routes.
func (h *Handler) RegisterRoutes(router corehttp.Router) {
	router.Post("/woo/sync/contacts", h.protect("contact:sync", h.syncContacts))
	router.Post("/woo/sync/orders", h.protect("order:sync", h.syncOrders))
	router.Post("/woo/sync/coupons", h.protect("coupon:sync", h.syncCoupons))
}

// syncContacts triggers manual contact sync behavior.
func (h *Handler) syncContacts(ctx corehttp.Context) error {
	targetEmail := strings.TrimSpace(ctx.Query("email", ""))

	var (
		summary *woocontactservice.SyncSummary
		err     error
	)
	if targetEmail != "" {
		summary, err = h.contactsService.SyncContactByEmail(ctx.Context(), "manual", targetEmail)
	} else {
		summary, err = h.contactsService.SyncContacts(ctx.Context(), "manual")
	}
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(summary)
}

// syncOrders triggers manual order sync behavior.
func (h *Handler) syncOrders(ctx corehttp.Context) error {
	rawOrderID := strings.TrimSpace(ctx.Query("id", ""))

	var (
		summary *wooorderservice.SyncSummary
		err     error
	)
	if rawOrderID != "" {
		orderID, parseErr := strconv.Atoi(rawOrderID)
		if parseErr != nil || orderID <= 0 {
			return h.mapError(wooorderservice.ErrInvalidOrderID)
		}

		summary, err = h.ordersService.SyncOrderByID(ctx.Context(), "manual", orderID)
	} else {
		summary, err = h.ordersService.SyncOrders(ctx.Context(), "manual")
	}
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(summary)
}

// syncCoupons triggers manual coupon sync behavior.
func (h *Handler) syncCoupons(ctx corehttp.Context) error {
	if h.couponsService == nil {
		return h.mapError(woocouponservice.ErrSyncDisabled)
	}

	summary, err := h.couponsService.SyncCoupons(ctx.Context(), "manual")
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
	if errors.Is(err, woocontactservice.ErrSyncDisabled) {
		return corehttp.NewAppError(503, "woocommerce_contacts_sync_disabled", err)
	}
	if errors.Is(err, woocontactservice.ErrInvalidEmail) {
		return corehttp.NewAppError(400, "invalid_contact_email", err)
	}
	if errors.Is(err, woocontactservice.ErrContactNotFound) {
		return corehttp.NewAppError(404, "woocommerce_contact_not_found", err)
	}
	if errors.Is(err, wooorderservice.ErrSyncDisabled) {
		return corehttp.NewAppError(503, "woocommerce_orders_sync_disabled", err)
	}
	if errors.Is(err, wooorderservice.ErrInvalidOrderID) {
		return corehttp.NewAppError(400, "invalid_order_id", err)
	}
	if errors.Is(err, wooorderservice.ErrOrderNotFound) {
		return corehttp.NewAppError(404, "woocommerce_order_not_found", err)
	}
	if errors.Is(err, woocontactservice.ErrIntegrationUnavailable) {
		return corehttp.NewAppError(503, "woocommerce_integration_unavailable", err)
	}
	if errors.Is(err, wooorderservice.ErrIntegrationUnavailable) {
		return corehttp.NewAppError(503, "woocommerce_integration_unavailable", err)
	}
	if errors.Is(err, woocouponservice.ErrSyncDisabled) {
		return corehttp.NewAppError(503, "woocommerce_coupons_sync_disabled", err)
	}
	if errors.Is(err, woocouponservice.ErrIntegrationUnavailable) || errors.Is(err, woocouponservice.ErrUpsertUnavailable) {
		return corehttp.NewAppError(503, "woocommerce_integration_unavailable", err)
	}

	return corehttp.NewAppError(500, "internal_server_error", err)
}
