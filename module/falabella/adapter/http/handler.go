package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	corehttp "mannaiah/module/core/http"
	brandservice "mannaiah/module/falabella/application/brand/service"
	productsyncservice "mannaiah/module/falabella/application/productsync/service"
	syncstatusservice "mannaiah/module/falabella/application/syncstatus/service"
	syncdomain "mannaiah/module/falabella/domain/sync"
	"mannaiah/module/falabella/port"
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

// SyncStatusService defines Falabella sync status use-case behavior required by HTTP handlers.
type SyncStatusService interface {
	// GetExecutionByID retrieves one sync execution by identifier.
	GetExecutionByID(ctx context.Context, executionID string) (*syncdomain.SyncExecution, error)
	// GetByFeedID retrieves a sync status entry by Falabella feed identifier.
	GetByFeedID(ctx context.Context, feedID string) (*syncdomain.SyncEntry, error)
	// GetByExecutionID retrieves child feed rows by execution identifier.
	GetByExecutionID(ctx context.Context, executionID string) ([]syncdomain.SyncEntry, error)
	// GetByProductID retrieves sync status entries by source product identifier.
	GetByProductID(ctx context.Context, productID string) ([]syncdomain.SyncEntry, error)
	// ResolveFeedStatus queries Falabella feed status and updates the entry resolution.
	ResolveFeedStatus(ctx context.Context, feedID string) (*syncstatusservice.ResolveResult, error)
}

// Handler defines HTTP route handlers for Falabella integration endpoints.
type Handler struct {
	// service defines Falabella brand service dependencies.
	service Service
	// productSyncService defines Falabella product-sync service dependencies.
	productSyncService ProductSyncService
	// syncStatusService defines optional Falabella sync status service dependencies.
	syncStatusService SyncStatusService
	// authorizer defines optional auth dependency for protected endpoints.
	authorizer Authorizer
	// imageTranscode defines optional image-transcoding endpoint configuration.
	imageTranscode ImageTranscodeConfig
}

// ImageTranscodeConfig defines image-transcoding endpoint behavior configuration values.
type ImageTranscodeConfig struct {
	// Enabled defines whether the image transcode endpoint should process requests.
	Enabled bool
	// AllowedSourcePrefixes defines optional source URL prefixes allowed for transcode requests.
	AllowedSourcePrefixes []string
	// RequestTimeout defines source image fetch timeout values.
	RequestTimeout time.Duration
	// MaxInputBytes defines maximum source payload bytes read before decode.
	MaxInputBytes int64
	// HTTPClient defines optional custom HTTP client dependencies used to fetch source images.
	HTTPClient *http.Client
}

// NewHandler creates Falabella HTTP handlers.
func NewHandler(service Service, productSyncService ProductSyncService, syncStatusServices ...SyncStatusService) (*Handler, error) {
	if service == nil {
		return nil, ErrNilService
	}
	if productSyncService == nil {
		return nil, ErrNilProductSyncService
	}

	var syncStatusService SyncStatusService
	if len(syncStatusServices) > 0 && syncStatusServices[0] != nil {
		syncStatusService = syncStatusServices[0]
	}

	return &Handler{
		service:            service,
		productSyncService: productSyncService,
		syncStatusService:  syncStatusService,
		imageTranscode: ImageTranscodeConfig{
			RequestTimeout: 15 * time.Second,
			MaxInputBytes:  20 << 20,
		},
	}, nil
}

// SetAuthorizer configures endpoint authentication and authorization dependencies.
func (h *Handler) SetAuthorizer(authorizer Authorizer) {
	if h == nil {
		return
	}

	h.authorizer = authorizer
}

// SetImageTranscodeConfig configures image-transcoding endpoint behavior.
func (h *Handler) SetImageTranscodeConfig(cfg ImageTranscodeConfig) {
	if h == nil {
		return
	}

	resolved := cfg
	if resolved.RequestTimeout <= 0 {
		resolved.RequestTimeout = 15 * time.Second
	}
	if resolved.MaxInputBytes <= 0 {
		resolved.MaxInputBytes = 20 << 20
	}
	if resolved.HTTPClient == nil {
		resolved.HTTPClient = &http.Client{Timeout: resolved.RequestTimeout}
	}

	allowed := make([]string, 0, len(resolved.AllowedSourcePrefixes))
	for _, prefix := range resolved.AllowedSourcePrefixes {
		trimmed := strings.TrimRight(strings.TrimSpace(prefix), "/")
		if trimmed == "" {
			continue
		}
		allowed = append(allowed, trimmed)
	}
	resolved.AllowedSourcePrefixes = allowed

	h.imageTranscode = resolved
}

// RegisterRoutes registers Falabella integration routes.
func (h *Handler) RegisterRoutes(router corehttp.Router) {
	router.Get("/falabella/images/transcoded", h.transcodeImage)
	router.Get("/falabella/brands", h.protect("products:read", h.getBrands))
	router.Post("/falabella/sync/products", h.protect("products:update", h.syncProducts))
	router.Post("/falabella/sync/products/:id", h.protect("products:update", h.syncProductByID))
	router.Get("/falabella/sync/status/feed/:feedId", h.protect("products:read", h.getSyncStatusByFeed))
	router.Get("/falabella/sync/status/execution/:executionId", h.protect("products:read", h.getSyncStatusExecution))
	router.Get("/falabella/sync/status/execution/:executionId/feeds", h.protect("products:read", h.getSyncStatusByExecution))
	router.Get("/falabella/sync/status/product/:productId", h.protect("products:read", h.getSyncStatusByProduct))
	router.Post("/falabella/sync/status/feed/:feedId/resolve", h.protect("products:update", h.resolveFeedStatus))
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

// getSyncStatusByFeed retrieves sync status by Falabella feed identifier.
func (h *Handler) getSyncStatusByFeed(ctx corehttp.Context) error {
	if h.syncStatusService == nil {
		return corehttp.NewAppError(503, "sync_status_unavailable", errors.New("sync status service is not configured"))
	}

	entry, err := h.syncStatusService.GetByFeedID(ctx.Context(), strings.TrimSpace(ctx.Params("feedId")))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(mapSyncEntryResponse(entry))
}

// getSyncStatusExecution retrieves one execution parent by identifier.
func (h *Handler) getSyncStatusExecution(ctx corehttp.Context) error {
	if h.syncStatusService == nil {
		return corehttp.NewAppError(503, "sync_status_unavailable", errors.New("sync status service is not configured"))
	}

	execution, err := h.syncStatusService.GetExecutionByID(ctx.Context(), strings.TrimSpace(ctx.Params("executionId")))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(mapSyncExecutionResponse(execution))
}

// getSyncStatusByExecution retrieves child feed rows by execution identifier.
func (h *Handler) getSyncStatusByExecution(ctx corehttp.Context) error {
	if h.syncStatusService == nil {
		return corehttp.NewAppError(503, "sync_status_unavailable", errors.New("sync status service is not configured"))
	}

	entries, err := h.syncStatusService.GetByExecutionID(ctx.Context(), strings.TrimSpace(ctx.Params("executionId")))
	if err != nil {
		return h.mapError(err)
	}

	response := make([]syncStatusEntryResponse, 0, len(entries))
	for i := range entries {
		response = append(response, mapSyncEntryResponse(&entries[i]))
	}

	return ctx.Status(200).JSON(response)
}

// getSyncStatusByProduct retrieves sync status entries by source product identifier.
func (h *Handler) getSyncStatusByProduct(ctx corehttp.Context) error {
	if h.syncStatusService == nil {
		return corehttp.NewAppError(503, "sync_status_unavailable", errors.New("sync status service is not configured"))
	}

	entries, err := h.syncStatusService.GetByProductID(ctx.Context(), strings.TrimSpace(ctx.Params("productId")))
	if err != nil {
		return h.mapError(err)
	}

	response := make([]syncStatusEntryResponse, 0, len(entries))
	for i := range entries {
		response = append(response, mapSyncEntryResponse(&entries[i]))
	}

	return ctx.Status(200).JSON(response)
}

// resolveFeedStatus resolves Falabella feed status and updates sync entry.
func (h *Handler) resolveFeedStatus(ctx corehttp.Context) error {
	if h.syncStatusService == nil {
		return corehttp.NewAppError(503, "sync_status_unavailable", errors.New("sync status service is not configured"))
	}

	result, err := h.syncStatusService.ResolveFeedStatus(ctx.Context(), strings.TrimSpace(ctx.Params("feedId")))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(result)
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
	if errors.Is(err, syncstatusservice.ErrInvalidFeedID) {
		return corehttp.NewAppError(400, "invalid_feed_id", err)
	}
	if errors.Is(err, syncstatusservice.ErrInvalidExecutionID) {
		return corehttp.NewAppError(400, "invalid_execution_id", err)
	}
	if errors.Is(err, syncstatusservice.ErrInvalidProductID) {
		return corehttp.NewAppError(400, "invalid_product_id", err)
	}
	if errors.Is(err, syncstatusservice.ErrFeedNotFinished) {
		return corehttp.NewAppError(409, "feed_not_finished", err)
	}
	if errors.Is(err, port.ErrSyncEntryNotFound) {
		return corehttp.NewAppError(404, "sync_entry_not_found", err)
	}
	if errors.Is(err, port.ErrSyncExecutionNotFound) {
		return corehttp.NewAppError(404, "sync_execution_not_found", err)
	}

	return corehttp.NewAppError(500, "internal_server_error", err)
}

// syncProductsRequest defines batch product-sync request payload values.
type syncProductsRequest struct {
	// IDs defines optional product IDs to synchronize.
	IDs []string `json:"ids"`
}

// syncStatusEntryResponse defines sync status entry response values.
type syncStatusEntryResponse struct {
	// ExecutionID defines parent execution identifier values.
	ExecutionID string `json:"executionId,omitempty"`
	// FeedID defines Falabella feed identifier values.
	FeedID string `json:"feedId"`
	// ProductID defines source product identifier values.
	ProductID string `json:"productId"`
	// SKU defines seller SKU values.
	SKU string `json:"sku"`
	// VariationIDs defines linked product variation identifier values.
	VariationIDs []string `json:"variationIds,omitempty"`
	// Step defines logical feed step values (product/image).
	Step string `json:"step,omitempty"`
	// Task defines high-level sync task category values (data/image).
	Task string `json:"task,omitempty"`
	// Action defines sync operation type values.
	Action string `json:"action"`
	// Status defines feed resolution status values.
	Status string `json:"status"`
	// SyncedAt defines sync submission timestamp values.
	SyncedAt string `json:"syncedAt"`
	// ResolvedAt defines optional feed resolution timestamp values.
	ResolvedAt string `json:"resolvedAt,omitempty"`
}

// syncStatusExecutionResponse defines parent sync execution response values.
type syncStatusExecutionResponse struct {
	// ExecutionID defines parent execution identifier values.
	ExecutionID string `json:"executionId"`
	// StartedAt defines execution start timestamp values.
	StartedAt string `json:"startedAt"`
}

// mapSyncEntryResponse maps domain sync entries to HTTP response values.
func mapSyncEntryResponse(entry *syncdomain.SyncEntry) syncStatusEntryResponse {
	if entry == nil {
		return syncStatusEntryResponse{}
	}

	response := syncStatusEntryResponse{
		ExecutionID:  entry.ExecutionID,
		FeedID:       entry.FeedID,
		ProductID:    entry.ProductID,
		SKU:          entry.SKU,
		VariationIDs: append([]string(nil), entry.VariationIDs...),
		Step:         entry.Step.String(),
		Task:         entry.Task.String(),
		Action:       entry.Action.String(),
		Status:       entry.Status.String(),
		SyncedAt:     entry.SyncedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if response.Task == "" {
		response.Task = entry.Step.Task().String()
	}
	if entry.ResolvedAt != nil {
		response.ResolvedAt = entry.ResolvedAt.UTC().Format("2006-01-02T15:04:05Z")
	}

	return response
}

// mapSyncExecutionResponse maps domain sync execution values to HTTP response values.
func mapSyncExecutionResponse(execution *syncdomain.SyncExecution) syncStatusExecutionResponse {
	if execution == nil {
		return syncStatusExecutionResponse{}
	}

	return syncStatusExecutionResponse{
		ExecutionID: execution.ExecutionID,
		StartedAt:   execution.StartedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}
