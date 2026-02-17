package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"

	corehttp "mannaiah/module/core/http"
	brandservice "mannaiah/module/falabella/application/brand/service"
	productsyncservice "mannaiah/module/falabella/application/productsync/service"
)

var (
	// ErrNilService is returned when service dependencies are nil.
	ErrNilService = errors.New("falabella brand service must not be nil")
	// ErrNilProductSyncService is returned when product-sync service dependencies are nil.
	ErrNilProductSyncService = errors.New("falabella product sync service must not be nil")
)

// Authorizer defines authentication and authorization behavior required by Falabella endpoints.
type Authorizer interface {
	// Require authenticates and authorizes requests using required permissions.
	Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
	// IsUnauthorized reports authentication errors.
	IsUnauthorized(err error) bool
	// IsForbidden reports authorization errors.
	IsForbidden(err error) bool
}

// Service defines Falabella brand use-case behavior required by HTTP handlers.
type Service interface {
	// GetBrands retrieves Falabella brand payload.
	GetBrands(ctx context.Context) ([]byte, error)
}

// ProductSyncService defines Falabella product-sync use-case behavior required by HTTP handlers.
type ProductSyncService interface {
	// SyncProduct syncs one product by identifier.
	SyncProduct(ctx context.Context, id string) (*productsyncservice.Summary, error)
	// SyncProducts syncs provided products or all products when ids are empty.
	SyncProducts(ctx context.Context, ids []string) (*productsyncservice.Summary, error)
}

// Handler defines HTTP route handlers for Falabella integration endpoints.
type Handler struct {
	// service defines Falabella brand service dependencies.
	service Service
	// productSyncService defines Falabella product-sync service dependencies.
	productSyncService ProductSyncService
	// authorizer defines optional auth dependency for protected endpoints.
	authorizer Authorizer
}

// NewHandler creates Falabella HTTP handlers.
func NewHandler(service Service, productSyncService ProductSyncService, authorizers ...Authorizer) (*Handler, error) {
	if service == nil {
		return nil, ErrNilService
	}
	if productSyncService == nil {
		return nil, ErrNilProductSyncService
	}

	var authorizer Authorizer
	if len(authorizers) > 0 {
		authorizer = authorizers[0]
	}

	return &Handler{service: service, productSyncService: productSyncService, authorizer: authorizer}, nil
}

// SetAuthorizer configures endpoint authentication and authorization dependencies.
func (h *Handler) SetAuthorizer(authorizer Authorizer) {
	if h == nil {
		return
	}

	h.authorizer = authorizer
}

// RegisterRoutes registers Falabella integration routes.
func (h *Handler) RegisterRoutes(router corehttp.Router) {
	router.Get("/falabella/brands", h.protect("products:read", h.getBrands))
	router.Post("/falabella/sync/products", h.protect("products:update", h.syncProducts))
	router.Post("/falabella/sync/products/:id", h.protect("products:update", h.syncProductByID))
}

// getBrands retrieves Falabella brands through integration service dependencies.
func (h *Handler) getBrands(ctx corehttp.Context) error {
	payload, err := h.service.GetBrands(ctx.Context())
	if err != nil {
		return h.mapError(err)
	}

	var body any
	if err := json.Unmarshal(payload, &body); err != nil {
		return corehttp.NewAppError(502, "falabella_invalid_payload", err)
	}

	return ctx.Status(200).JSON(body)
}

// syncProducts syncs one or many products to Falabella.
func (h *Handler) syncProducts(ctx corehttp.Context) error {
	request := syncProductsRequest{}
	if shouldParseBody(ctx) {
		if err := ctx.BodyParser(&request); err != nil && !errors.Is(err, io.EOF) {
			return corehttp.NewAppError(400, "invalid_body", err)
		}
	}

	summary, err := h.productSyncService.SyncProducts(ctx.Context(), request.IDs)
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(summary)
}

// shouldParseBody reports whether request payload parsing is required.
func shouldParseBody(ctx corehttp.Context) bool {
	contentLength := strings.TrimSpace(ctx.GetHeader("Content-Length"))
	if contentLength == "" {
		return false
	}
	length, err := strconv.ParseInt(contentLength, 10, 64)
	if err != nil {
		return true
	}

	return length > 0
}

// syncProductByID syncs one product to Falabella.
func (h *Handler) syncProductByID(ctx corehttp.Context) error {
	summary, err := h.productSyncService.SyncProduct(ctx.Context(), strings.TrimSpace(ctx.Params("id")))
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
	if errors.Is(err, brandservice.ErrIntegrationUnavailable) {
		return corehttp.NewAppError(503, "falabella_integration_unavailable", err)
	}
	if errors.Is(err, productsyncservice.ErrIntegrationUnavailable) {
		return corehttp.NewAppError(503, "falabella_integration_unavailable", err)
	}
	if errors.Is(err, productsyncservice.ErrInvalidProductID) {
		return corehttp.NewAppError(400, "invalid_product_id", err)
	}

	return corehttp.NewAppError(500, "internal_server_error", err)
}

// syncProductsRequest defines batch product-sync request payload values.
type syncProductsRequest struct {
	// IDs defines optional product IDs to synchronize.
	IDs []string `json:"ids"`
}
