package http

import (
	"context"
	"errors"
	"strconv"
	"strings"

	corehttp "mannaiah/module/core/http"
	"mannaiah/module/segment/application"
	"mannaiah/module/segment/domain"
)

var (
	// ErrNilService is returned when nil service dependencies are provided.
	ErrNilService = errors.New("segment service must not be nil")
)

// Authorizer defines authentication and authorization behavior required by segment endpoints.
type Authorizer interface {
	// Require authenticates and authorizes requests using required permissions.
	Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
	// IsUnauthorized reports authentication errors.
	IsUnauthorized(err error) bool
	// IsForbidden reports authorization errors.
	IsForbidden(err error) bool
}

// Service defines segment use-case behavior required by HTTP handlers.
type Service interface {
	// Create persists segment rows.
	Create(ctx context.Context, command application.CreateCommand) (*domain.Segment, error)
	// Get retrieves one segment by id.
	Get(ctx context.Context, id string) (*domain.Segment, error)
	// List retrieves paged segment rows.
	List(ctx context.Context, page int, limit int) (*application.ListResult, error)
	// Update persists segment row updates.
	Update(ctx context.Context, id string, command application.UpdateCommand) (*domain.Segment, error)
	// Delete removes one segment by id.
	Delete(ctx context.Context, id string) error
	// Resolve resolves contact ids for one segment.
	Resolve(ctx context.Context, id string, page int, limit int) (*application.ResolveResult, error)
	// Count resolves contact count for one segment.
	Count(ctx context.Context, id string) (int64, error)
}

// Handler defines HTTP route handlers for segment endpoints.
type Handler struct {
	// service defines segment use-case dependencies.
	service Service
	// authorizer defines optional auth dependencies.
	authorizer Authorizer
}

// createRequest defines create request payload values.
type createRequest struct {
	// Name defines segment names.
	Name string `json:"name"`
	// Slug defines segment slugs.
	Slug string `json:"slug"`
	// Channel defines target channel values.
	Channel string `json:"channel"`
	// Filters defines filter DSL values.
	Filters []domain.Filter `json:"filters"`
}

// updateRequest defines update request payload values.
type updateRequest struct {
	// Name defines optional segment names.
	Name *string `json:"name"`
	// Slug defines optional segment slugs.
	Slug *string `json:"slug"`
	// Channel defines optional target channel values.
	Channel *string `json:"channel"`
	// Filters defines optional filter DSL values.
	Filters *[]domain.Filter `json:"filters"`
}

// NewHandler creates segment HTTP handlers.
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

// RegisterRoutes registers segment routes.
func (h *Handler) RegisterRoutes(router corehttp.Router) {
	router.Post("/segments", h.protect("marketing:manage", h.create))
	router.Get("/segments", h.protect("marketing:manage", h.list))
	router.Get("/segments/:id", h.protect("marketing:manage", h.get))
	router.Patch("/segments/:id", h.protect("marketing:manage", h.update))
	router.Delete("/segments/:id", h.protect("marketing:manage", h.remove))
	router.Post("/segments/:id/resolve", h.protect("marketing:manage", h.resolve))
	router.Get("/segments/:id/count", h.protect("marketing:manage", h.count))
}

// create handles segment create requests.
func (h *Handler) create(ctx corehttp.Context) error {
	request := createRequest{}
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	segment, err := h.service.Create(ctx.Context(), application.CreateCommand{
		Name:    request.Name,
		Slug:    request.Slug,
		Channel: request.Channel,
		Filters: request.Filters,
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(201).JSON(segment)
}

// list handles segment list requests.
func (h *Handler) list(ctx corehttp.Context) error {
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	limit, _ := strconv.Atoi(ctx.Query("limit", "20"))

	result, err := h.service.List(ctx.Context(), page, limit)
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(result)
}

// get handles segment by-id requests.
func (h *Handler) get(ctx corehttp.Context) error {
	segment, err := h.service.Get(ctx.Context(), strings.TrimSpace(ctx.Params("id")))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(segment)
}

// update handles segment update requests.
func (h *Handler) update(ctx corehttp.Context) error {
	request := updateRequest{}
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	segment, err := h.service.Update(ctx.Context(), strings.TrimSpace(ctx.Params("id")), application.UpdateCommand{
		Name:    request.Name,
		Slug:    request.Slug,
		Channel: request.Channel,
		Filters: request.Filters,
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(segment)
}

// remove handles segment delete requests.
func (h *Handler) remove(ctx corehttp.Context) error {
	if err := h.service.Delete(ctx.Context(), strings.TrimSpace(ctx.Params("id"))); err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(map[string]string{"status": "deleted"})
}

// resolve handles segment resolution requests.
func (h *Handler) resolve(ctx corehttp.Context) error {
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	limit, _ := strconv.Atoi(ctx.Query("limit", "1000"))

	result, err := h.service.Resolve(ctx.Context(), strings.TrimSpace(ctx.Params("id")), page, limit)
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(result)
}

// count handles segment resolution count requests.
func (h *Handler) count(ctx corehttp.Context) error {
	count, err := h.service.Count(ctx.Context(), strings.TrimSpace(ctx.Params("id")))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(map[string]any{
		"segmentId": strings.TrimSpace(ctx.Params("id")),
		"count":     count,
	})
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
	if errors.Is(err, domain.ErrInvalidID) || errors.Is(err, domain.ErrInvalidName) || errors.Is(err, domain.ErrInvalidSlug) || errors.Is(err, domain.ErrInvalidFilter) {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}
	if errors.Is(err, application.ErrResolverUnavailable) {
		return corehttp.NewAppError(503, "segment_backend_unavailable", err)
	}
	if errors.Is(err, domain.ErrNotFound) {
		return corehttp.NewAppError(404, "segment_not_found", err)
	}

	return corehttp.NewAppError(500, "internal_server_error", err)
}
