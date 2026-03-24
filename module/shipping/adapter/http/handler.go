package http

import (
	"context"
	"errors"

	corehttp "mannaiah/module/core/http"
	dispatchservice "mannaiah/module/shipping/application/dispatch/service"
	markservice "mannaiah/module/shipping/application/mark/service"
	quotationservice "mannaiah/module/shipping/application/quotation/service"
	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

var (
	// ErrNilHandlerDependencies is returned when required services are nil.
	ErrNilHandlerDependencies = errors.New("shipping handler dependencies must not be nil")
)

// Authorizer defines authentication and authorization behavior required by shipping endpoints.
type Authorizer interface {
	// Require authenticates and authorizes requests using required permissions.
	Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
	// IsUnauthorized reports authentication errors.
	IsUnauthorized(err error) bool
	// IsForbidden reports authorization errors.
	IsForbidden(err error) bool
	// Subject resolves the caller subject from the authorization header.
	// Returns "system" for dev-bypass tokens or when authentication fails.
	Subject(ctx context.Context, authorizationHeader string) string
}

// QuotationService defines quotation behavior required by HTTP handlers.
type QuotationService interface {
	// Quote requests one freight quotation.
	Quote(ctx context.Context, command quotationservice.QuoteCommand) (*domain.QuotationResult, error)
	// ListByOrderID lists quotation records for one order.
	ListByOrderID(ctx context.Context, orderID string) ([]port.QuotationRecord, error)
}

// MarkService defines shipping mark behavior required by HTTP handlers.
type MarkService interface {
	// Generate creates one shipping mark.
	Generate(ctx context.Context, command markservice.GenerateCommand) (*domain.ShippingMark, error)
	// Get resolves one shipping mark by id.
	Get(ctx context.Context, id string) (*domain.ShippingMark, error)
	// List resolves shipping marks with filters and pagination.
	List(ctx context.Context, query markservice.ListQuery) ([]domain.ShippingMark, int64, error)
	// Void voids one shipping mark.
	Void(ctx context.Context, id string, reason string) (*domain.ShippingMark, error)
}

// DispatchService defines dispatch batch behavior required by HTTP handlers.
type DispatchService interface {
	// Create creates one dispatch batch.
	Create(ctx context.Context, command dispatchservice.CreateBatchCommand) (*domain.DispatchBatch, error)
	// Get resolves one dispatch batch by id.
	Get(ctx context.Context, id string) (*domain.DispatchBatch, error)
	// List resolves dispatch batches with filters and pagination.
	List(ctx context.Context, query dispatchservice.ListQuery) ([]domain.DispatchBatch, int64, error)
	// DraftMark creates one QUOTED draft mark and assigns it to an open batch.
	DraftMark(ctx context.Context, command dispatchservice.DraftMarkCommand) (*domain.ShippingMark, error)
	// RemoveDraftMark removes one QUOTED draft mark from a batch and sets it to REMOVED.
	RemoveDraftMark(ctx context.Context, batchID string, markID string) (*domain.DispatchBatch, error)
	// Close closes one dispatch batch.
	Close(ctx context.Context, batchID string) (*domain.DispatchBatch, error)
}

// TrackingService defines tracking behavior required by HTTP handlers.
type TrackingService interface {
	// Get resolves tracking history by carrier and tracking number.
	Get(ctx context.Context, carrierID string, trackingNumber string) (*domain.TrackingHistory, error)
}

// CarrierService defines carrier listing behavior required by HTTP handlers.
type CarrierService interface {
	// List returns available carriers.
	List(ctx context.Context) ([]domain.Carrier, error)
	// Get resolves one carrier by id.
	Get(ctx context.Context, id string) (*domain.Carrier, error)
}

// Handler defines HTTP route handlers for shipping endpoints.
type Handler struct {
	// quotations defines quotation service dependencies.
	quotations QuotationService
	// marks defines shipping mark service dependencies.
	marks MarkService
	// batches defines dispatch batch service dependencies.
	batches DispatchService
	// tracking defines tracking service dependencies.
	tracking TrackingService
	// carriers defines carrier listing dependencies.
	carriers CarrierService
	// authorizer defines optional auth dependencies.
	authorizer Authorizer
}

// NewHandler creates shipping HTTP handlers.
func NewHandler(quotations QuotationService, marks MarkService, batches DispatchService, tracking TrackingService, carriers CarrierService, authorizers ...Authorizer) (*Handler, error) {
	if quotations == nil || marks == nil || batches == nil || tracking == nil || carriers == nil {
		return nil, ErrNilHandlerDependencies
	}

	var authorizer Authorizer
	if len(authorizers) > 0 {
		authorizer = authorizers[0]
	}

	return &Handler{quotations: quotations, marks: marks, batches: batches, tracking: tracking, carriers: carriers, authorizer: authorizer}, nil
}

// SetAuthorizer configures endpoint authentication and permission dependencies.
func (h *Handler) SetAuthorizer(authorizer Authorizer) {
	if h == nil {
		return
	}

	h.authorizer = authorizer
}

// RegisterRoutes registers shipping routes.
func (h *Handler) RegisterRoutes(router corehttp.Router) {
	router.Post("/shipping/quotations", h.protect("marketing:manage", h.createQuotation))
	router.Get("/shipping/quotations", h.protect("marketing:manage", h.listQuotations))
	router.Post("/shipping/marks", h.protect("marketing:manage", h.createMark))
	router.Get("/shipping/marks/:id", h.protect("marketing:manage", h.getMark))
	router.Get("/shipping/marks", h.protect("marketing:manage", h.listMarks))
	router.Patch("/shipping/marks/:id/void", h.protect("marketing:manage", h.voidMark))
	router.Post("/shipping/batches", h.protect("marketing:manage", h.createBatch))
	router.Get("/shipping/batches/:id", h.protect("marketing:manage", h.getBatch))
	router.Get("/shipping/batches", h.protect("marketing:manage", h.listBatches))
	router.Post("/shipping/batches/:id/marks", h.protect("marketing:manage", h.addBatchMark))
	router.Delete("/shipping/batches/:id/marks/:markID", h.protect("marketing:manage", h.removeBatchMark))
	router.Patch("/shipping/batches/:id/close", h.protect("marketing:manage", h.closeBatch))
	router.Get("/shipping/tracking/:trackingNumber", h.protect("marketing:manage", h.getTracking))
	router.Get("/shipping/carriers", h.protect("marketing:manage", h.listCarriers))
	router.Get("/shipping/carriers/:id", h.protect("marketing:manage", h.getCarrier))
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
	if errors.Is(err, domain.ErrInvalidID) || errors.Is(err, domain.ErrInvalidCarrierID) || errors.Is(err, domain.ErrInvalidShipmentMode) {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}
	if errors.Is(err, domain.ErrCarrierNotSupported) {
		return corehttp.NewAppError(400, "carrier_not_supported", err)
	}
	if errors.Is(err, domain.ErrQuotationNotSupported) {
		return corehttp.NewAppError(400, "quotation_not_supported", err)
	}
	if errors.Is(err, domain.ErrTrackingNotSupported) {
		return corehttp.NewAppError(400, "tracking_not_supported", err)
	}
	if errors.Is(err, domain.ErrInsufficientBalance) {
		return corehttp.NewAppError(409, "insufficient_balance", err)
	}
	if errors.Is(err, domain.ErrBatchClosed) {
		return corehttp.NewAppError(409, "batch_closed", err)
	}
	if errors.Is(err, domain.ErrBatchCarrierMismatch) {
		return corehttp.NewAppError(409, "batch_carrier_mismatch", err)
	}
	if errors.Is(err, domain.ErrBatchMarkStatusMismatch) {
		return corehttp.NewAppError(409, "batch_mark_status_mismatch", err)
	}
	if errors.Is(err, domain.ErrMarkNotDraft) {
		return corehttp.NewAppError(409, "mark_not_draft", err)
	}
	if errors.Is(err, domain.ErrNotFound) {
		return corehttp.NewAppError(404, "shipping_resource_not_found", err)
	}

	return corehttp.NewAppError(500, "internal_server_error", err)
}
