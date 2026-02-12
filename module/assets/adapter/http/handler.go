package http

import (
	"context"
	"errors"
	"io"
	"strconv"
	"strings"

	assetsapplication "mannaiah/module/assets/application"
	"mannaiah/module/assets/domain"
	"mannaiah/module/assets/port"
	corehttp "mannaiah/module/core/http"
)

var (
	// ErrNilService is returned when service dependencies are nil.
	ErrNilService = errors.New("assets service must not be nil")
)

// Authorizer defines authentication and authorization behavior required by asset endpoints.
type Authorizer interface {
	// Require authenticates and authorizes requests using required permissions.
	Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
	// IsUnauthorized reports authentication errors.
	IsUnauthorized(err error) bool
	// IsForbidden reports authorization errors.
	IsForbidden(err error) bool
}

// Handler defines HTTP route handlers for assets.
type Handler struct {
	// service defines asset use-case dependencies.
	service assetsapplication.Service
	// authorizer defines optional endpoint auth dependencies.
	authorizer Authorizer
}

// updateRequest defines request payload for asset update operations.
type updateRequest struct {
	// Name defines target asset names.
	Name string `json:"name"`
}

// listResponse defines response payload for paginated asset listing.
type listResponse struct {
	// Data defines listed rows.
	Data []domain.Asset `json:"data"`
	// Meta defines pagination metadata.
	Meta listResponseMeta `json:"meta"`
}

// listResponseMeta defines pagination metadata payload values.
type listResponseMeta struct {
	// Page defines current page numbers.
	Page int `json:"page"`
	// Total defines total rows.
	Total int64 `json:"total"`
	// Limit defines page size values.
	Limit int `json:"limit"`
}

// deleteResponse defines delete response payload values.
type deleteResponse struct {
	// Status defines operation status values.
	Status string `json:"status"`
}

// NewHandler creates asset HTTP handlers.
func NewHandler(service assetsapplication.Service, authorizers ...Authorizer) (*Handler, error) {
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

// RegisterRoutes registers asset CRUD endpoints.
func (h *Handler) RegisterRoutes(router corehttp.Router) {
	router.Post("/assets", h.protect("assets:create", h.create))
	router.Get("/assets", h.protect("assets:read", h.findAll))
	router.Get("/assets/:id", h.protect("assets:read", h.findOne))
	router.Patch("/assets/:id", h.protect("assets:update", h.update))
	router.Delete("/assets/:id", h.protect("assets:delete", h.remove))
}

// create handles asset upload endpoints.
func (h *Handler) create(ctx corehttp.Context) error {
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		return corehttp.NewAppError(400, "file_required", assetsapplication.ErrFileRequired)
	}

	file, err := fileHeader.Open()
	if err != nil {
		return corehttp.NewAppError(400, "invalid_file", err)
	}
	defer func() {
		_ = file.Close()
	}()

	body, err := io.ReadAll(file)
	if err != nil {
		return corehttp.NewAppError(400, "invalid_file", err)
	}

	name := strings.TrimSpace(ctx.FormValue("name"))
	mimeType := strings.TrimSpace(fileHeader.Header.Get("Content-Type"))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	entity, createErr := h.service.Create(ctx.Context(), assetsapplication.CreateCommand{
		Name:         name,
		OriginalName: fileHeader.Filename,
		MimeType:     mimeType,
		Size:         fileHeader.Size,
		Body:         body,
	})
	if createErr != nil {
		return h.mapError(createErr)
	}

	return ctx.Status(201).JSON(entity)
}

// findAll handles paginated list endpoints.
func (h *Handler) findAll(ctx corehttp.Context) error {
	page, err := parseIntQuery(ctx, "page", 1)
	if err != nil {
		return corehttp.NewAppError(400, "invalid_page", err)
	}
	limit, err := parseIntQuery(ctx, "limit", 10)
	if err != nil {
		return corehttp.NewAppError(400, "invalid_limit", err)
	}
	filters := strings.TrimSpace(ctx.Query("filters"))

	result, listErr := h.service.List(ctx.Context(), assetsapplication.ListQuery{
		Page:    page,
		Limit:   limit,
		Filters: filters,
	})
	if listErr != nil {
		return h.mapError(listErr)
	}

	return ctx.Status(200).JSON(listResponse{
		Data: result.Data,
		Meta: listResponseMeta{Page: result.Page, Total: result.Total, Limit: result.Limit},
	})
}

// findOne handles asset-by-id retrieval endpoints.
func (h *Handler) findOne(ctx corehttp.Context) error {
	entity, err := h.service.Get(ctx.Context(), ctx.Params("id"))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(entity)
}

// update handles asset name update endpoints.
func (h *Handler) update(ctx corehttp.Context) error {
	var request updateRequest
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	entity, err := h.service.UpdateName(ctx.Context(), ctx.Params("id"), request.Name)
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(entity)
}

// remove handles asset delete endpoints.
func (h *Handler) remove(ctx corehttp.Context) error {
	if err := h.service.Delete(ctx.Context(), ctx.Params("id")); err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(deleteResponse{Status: "deleted"})
}

// protect wraps handlers with optional auth checks.
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

// mapError maps service/domain/repository/auth errors to HTTP-layer app errors.
func (h *Handler) mapError(err error) error {
	if h != nil && h.authorizer != nil {
		if h.authorizer.IsUnauthorized(err) {
			return corehttp.NewAppError(401, "unauthorized", err)
		}
		if h.authorizer.IsForbidden(err) {
			return corehttp.NewAppError(403, "forbidden", err)
		}
	}
	if errors.Is(err, assetsapplication.ErrStorageUnavailable) {
		return corehttp.NewAppError(503, "storage_unavailable", err)
	}
	if errors.Is(err, assetsapplication.ErrInvalidID) {
		return corehttp.NewAppError(400, "invalid_asset_id", err)
	}
	if errors.Is(err, assetsapplication.ErrInvalidName) {
		return corehttp.NewAppError(400, "invalid_asset_name", err)
	}
	if errors.Is(err, assetsapplication.ErrFileRequired) {
		return corehttp.NewAppError(400, "file_required", err)
	}
	if errors.Is(err, assetsapplication.ErrFileTooLarge) {
		return corehttp.NewAppError(400, "file_too_large", err)
	}
	if errors.Is(err, domain.ErrKeyRequired) ||
		errors.Is(err, domain.ErrOriginalNameRequired) ||
		errors.Is(err, domain.ErrMimeTypeRequired) ||
		errors.Is(err, domain.ErrInvalidSize) {
		return corehttp.NewAppError(400, "invalid_asset", err)
	}
	if errors.Is(err, port.ErrNotFound) {
		return corehttp.NewAppError(404, "asset_not_found", err)
	}

	return corehttp.NewAppError(500, "internal_server_error", err)
}

// parseIntQuery parses integer query params with defaults.
func parseIntQuery(ctx corehttp.Context, key string, fallback int) (int, error) {
	value := strings.TrimSpace(ctx.Query(key))
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}

	return parsed, nil
}
