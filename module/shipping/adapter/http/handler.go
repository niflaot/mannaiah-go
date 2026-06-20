package http

import (
	"context"
	"errors"

	corehttp "mannaiah/module/core/http"
	dispatchservice "mannaiah/module/shipping/application/dispatch/service"
	markservice "mannaiah/module/shipping/application/mark/service"
	quotationservice "mannaiah/module/shipping/application/quotation/service"
	trackingservice "mannaiah/module/shipping/application/tracking/service"
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
	// QuoteFromOrder builds packages from order products and requests a freight quotation.
	QuoteFromOrder(ctx context.Context, command quotationservice.QuoteFromOrderCommand) (*domain.QuotationResult, error)
	// OrderPackagingFromOrder builds package allocation from order products without carrier quotation calls.
	OrderPackagingFromOrder(ctx context.Context, command quotationservice.QuoteFromOrderCommand) (*quotationservice.OrderPackagingResult, error)
	// ListByOrderID lists quotation records for one order.
	ListByOrderID(ctx context.Context, orderID string) ([]port.QuotationRecord, error)
	// GetLatestByOrderAndCarrier returns the most recent non-expired quotation for an order and carrier.
	GetLatestByOrderAndCarrier(ctx context.Context, orderID string, carrierID string) (*port.QuotationRecord, error)
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
	// QueryDispatch resolves the dispatch provisioning status for one order.
	QueryDispatch(ctx context.Context, query markservice.DispatchQuery) (*markservice.DispatchResult, error)
	// Related resolves related shipping marks by mark identifier.
	Related(ctx context.Context, id string) ([]domain.ShippingMark, error)
	// RotulusDocument builds one rotulus PDF document for one mark.
	RotulusDocument(ctx context.Context, id string) ([]byte, error)
	// MarkDocument downloads one shipping label PDF for one mark.
	MarkDocument(ctx context.Context, id string) ([]byte, error)
	// BatchAllMarksDocument downloads and merges all shipping label PDFs for marks in a batch.
	BatchAllMarksDocument(ctx context.Context, batchID string) ([]byte, error)
	// BatchAllRotulusDocument builds one PDF with all rotulus for marks in a batch, two per page.
	BatchAllRotulusDocument(ctx context.Context, batchID string) ([]byte, error)
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
	// CreateBatchMarkFromQuotation creates one batch mark from one quotation id.
	CreateBatchMarkFromQuotation(ctx context.Context, command dispatchservice.CreateBatchMarkFromQuotationCommand) (*domain.ShippingMark, error)
	// CreateBatchMark creates one batch mark as draft (quoted) or direct (materialized immediately).
	CreateBatchMark(ctx context.Context, command dispatchservice.CreateBatchMarkCommand) (*domain.ShippingMark, error)
	// UpdateDraftMark completes one existing manual QUOTED draft mark inside an open batch.
	UpdateDraftMark(ctx context.Context, command dispatchservice.UpdateDraftMarkCommand) (*domain.ShippingMark, error)
	// RemoveDraftMark removes one QUOTED draft mark from a batch and sets it to REMOVED.
	RemoveDraftMark(ctx context.Context, batchID string, markID string) (*domain.DispatchBatch, error)
	// Close closes one dispatch batch.
	Close(ctx context.Context, batchID string) (*domain.DispatchBatch, error)
	// ManifestDocument builds one merged manifest PDF document for a closed batch.
	ManifestDocument(ctx context.Context, batchID string) ([]byte, error)
	// ChecklistDocument builds one checklist PDF document for an open batch.
	ChecklistDocument(ctx context.Context, batchID string) ([]byte, error)
}

// TrackingService defines tracking behavior required by HTTP handlers.
type TrackingService interface {
	// Get resolves tracking history by carrier and tracking number.
	Get(ctx context.Context, carrierID string, trackingNumber string) (*domain.TrackingHistory, error)
	// List resolves paginated tracking summaries.
	List(ctx context.Context, query trackingservice.ListQuery) ([]trackingservice.ListItem, int64, error)
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
	router.Post("/shipping/quotations", h.protect("shipping:quotations", h.createQuotation))
	router.Get("/shipping/quotations", h.protect("shipping:quotations", h.listQuotations))
	router.Post("/shipping/quotations/order", h.protect("shipping:quotations", h.quoteFromOrder))
	router.Post("/shipping/quotations/order-packaging", h.protect("shipping:quotations", h.quoteOrderPackaging))
	router.Get("/shipping/quotations/order/:identifier", h.protect("shipping:quotations", h.getOrderQuotation))
	router.Post("/shipping/marks", h.protect("shipping:generate", h.createMark))
	router.Get("/shipping/marks/:id", h.protect("shipping:quotations", h.getMark))
	router.Get("/shipping/marks/:id/related", h.protect("shipping:quotations", h.listRelatedMarks))
	router.Get("/shipping/marks/:id/document", h.protect("shipping:generate", h.markDocument))
	router.Get("/shipping/marks/:id/rotulus-document", h.protectAny([]string{"shipping:generate", "shipping:quotations", "order:view"}, h.rotulusDocument))
	router.Get("/shipping/marks", h.protect("shipping:quotations", h.listMarks))
	router.Patch("/shipping/marks/:id/void", h.protect("shipping:manage", h.voidMark))
	router.Get("/shipping/orders/:orderID/dispatch", h.protect("shipping:quotations", h.getOrderDispatch))
	router.Post("/shipping/batches", h.protect("shipping:generate", h.createBatch))
	router.Get("/shipping/batches/:id", h.protect("shipping:quotations", h.getBatch))
	router.Get("/shipping/batches", h.protect("shipping:quotations", h.listBatches))
	router.Post("/shipping/batches/:id/marks", h.protect("shipping:generate", h.addBatchMark))
	router.Post("/shipping/batches/marks", h.protect("shipping:generate", h.createBatchMark))
	router.Patch("/shipping/batches/:id/marks/:markID", h.protect("shipping:generate", h.updateBatchMark))
	router.Delete("/shipping/batches/:id/marks/:markID", h.protect("shipping:generate", h.removeBatchMark))
	router.Patch("/shipping/batches/:id/close", h.protect("shipping:generate", h.closeBatch))
	router.Get("/shipping/batches/:id/manifest-document", h.protect("shipping:generate", h.batchManifestDocument))
	router.Get("/shipping/batches/:id/checklist-document", h.protect("shipping:generate", h.batchChecklistDocument))
	router.Get("/shipping/batches/:id/marks-all-document", h.protect("shipping:generate", h.batchAllMarksDocument))
	router.Get("/shipping/batches/:id/rotulus-all-document", h.protectAny([]string{"shipping:generate", "shipping:quotations", "order:view"}, h.batchAllRotulusDocument))
	router.Get("/shipping/tracking", h.protectAny([]string{"shipping:quotations", "shipping:generate", "shipping:manage"}, h.listTracking))
	router.Get("/shipping/tracking/:trackingNumber", h.protectAny([]string{"shipping:quotations", "shipping:generate", "shipping:manage"}, h.getTracking))
	router.Get("/shipping/carriers", h.protect("shipping:quotations", h.listCarriers))
	router.Get("/shipping/carriers/:id", h.protect("shipping:quotations", h.getCarrier))
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

// protectAny wraps endpoint handlers with optional authentication and "any-of" permission checks.
func (h *Handler) protectAny(permissions []string, next corehttp.Handler) corehttp.Handler {
	if h == nil || h.authorizer == nil || len(permissions) == 0 {
		return next
	}

	return func(ctx corehttp.Context) error {
		authorizationHeader := ctx.GetHeader("Authorization")
		var firstErr error
		var unauthorizedErr error
		var forbiddenErr error
		for _, permission := range permissions {
			err := h.authorizer.Require(ctx.Context(), authorizationHeader, permission)
			if err == nil {
				return next(ctx)
			}
			if firstErr == nil {
				firstErr = err
			}
			if h.authorizer.IsUnauthorized(err) && unauthorizedErr == nil {
				unauthorizedErr = err
			}
			if h.authorizer.IsForbidden(err) && forbiddenErr == nil {
				forbiddenErr = err
			}
		}
		if forbiddenErr != nil {
			return h.mapError(forbiddenErr)
		}
		if unauthorizedErr != nil {
			return h.mapError(unauthorizedErr)
		}

		return h.mapError(firstErr)
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
	if errors.Is(err, domain.ErrInvalidCityCode) {
		return corehttp.NewAppError(400, "invalid_city_code", err)
	}
	if errors.Is(err, domain.ErrNoValidProducts) {
		return corehttp.NewAppError(400, "no_valid_products", err)
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
	if errors.Is(err, domain.ErrInvalidBatchStatus) {
		return corehttp.NewAppError(409, "batch_status_invalid", err)
	}
	if errors.Is(err, domain.ErrBatchCarrierMismatch) {
		return corehttp.NewAppError(409, "batch_carrier_mismatch", err)
	}
	if errors.Is(err, domain.ErrBatchMarkStatusMismatch) {
		return corehttp.NewAppError(409, "batch_mark_status_mismatch", err)
	}
	if errors.Is(err, domain.ErrBatchOpenForCarrier) {
		return corehttp.NewAppError(409, "batch_open_for_carrier", err)
	}
	if errors.Is(err, domain.ErrMarkNotDraft) {
		return corehttp.NewAppError(409, "mark_not_draft", err)
	}
	if errors.Is(err, domain.ErrManualDraftIncomplete) {
		return corehttp.NewAppError(409, "manual_draft_incomplete", err)
	}
	if errors.Is(err, domain.ErrManualDraftUpdateNotSupported) {
		return corehttp.NewAppError(409, "manual_draft_update_not_supported", err)
	}
	if errors.Is(err, domain.ErrNotFound) {
		return corehttp.NewAppError(404, "shipping_resource_not_found", err)
	}
	var guardrailErr *domain.GuardrailViolationError
	if errors.As(err, &guardrailErr) {
		return corehttp.NewAppError(500, "shipping_guardrail_violation", err)
	}

	return corehttp.NewAppError(500, "internal_server_error", err)
}
