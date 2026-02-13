package http

import (
	"context"
	"encoding/json"
	"errors"
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

// listResponse defines response payload for paginated asset listing.
type listResponse struct {
	// Data defines listed rows.
	Data []domain.Asset `json:"data"`
	// Meta defines pagination metadata.
	Meta listResponseMeta `json:"meta"`
}

// folderListResponse defines response payload for paginated folder listing.
type folderListResponse struct {
	// Data defines listed rows.
	Data []domain.Folder `json:"data"`
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

// RegisterRoutes registers asset and folder CRUD endpoints.
func (h *Handler) RegisterRoutes(router corehttp.Router) {
	router.Post("/assets/folders", h.protect("assets:create", h.createFolder))
	router.Get("/assets/folders", h.protect("assets:read", h.findFolders))
	router.Get("/assets/folders/:id", h.protect("assets:read", h.findFolderByID))
	router.Patch("/assets/folders/:id", h.protect("assets:update", h.updateFolder))
	router.Delete("/assets/folders/:id", h.protect("assets:delete", h.deleteFolder))

	router.Post("/assets", h.protect("assets:create", h.createAsset))
	router.Get("/assets", h.protect("assets:read", h.findAssets))
	router.Get("/assets/:id", h.protect("assets:read", h.findAssetByID))
	router.Patch("/assets/:id", h.protect("assets:update", h.updateAsset))
	router.Delete("/assets/:id", h.protect("assets:delete", h.deleteAsset))
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
	if errors.Is(err, assetsapplication.ErrInvalidFolderID) {
		return corehttp.NewAppError(400, "invalid_folder_id", err)
	}
	if errors.Is(err, assetsapplication.ErrInvalidFolderName) {
		return corehttp.NewAppError(400, "invalid_folder_name", err)
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
		errors.Is(err, domain.ErrInvalidSize) ||
		errors.Is(err, domain.ErrFolderNameRequired) ||
		errors.Is(err, domain.ErrFolderSlugInvalid) ||
		errors.Is(err, domain.ErrTooManyTags) ||
		errors.Is(err, domain.ErrInvalidTagName) ||
		errors.Is(err, domain.ErrInvalidTagColor) ||
		errors.Is(err, domain.ErrDuplicatedTags) ||
		errors.Is(err, domain.ErrInvalidMetadata) {
		return corehttp.NewAppError(400, "invalid_asset", err)
	}
	if errors.Is(err, port.ErrFolderNotFound) {
		return corehttp.NewAppError(404, "asset_folder_not_found", err)
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

// parseJSONField decodes optional JSON form-field values.
func parseJSONField[T any](raw string) (*T, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	decoded := new(T)
	if err := json.Unmarshal([]byte(trimmed), decoded); err != nil {
		return nil, err
	}

	return decoded, nil
}
