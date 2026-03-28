package tag

import (
	"context"
	"errors"
	"strconv"
	"strings"

	corehttp "mannaiah/module/core/http"
	tagapplication "mannaiah/module/products/application/tag"
	tagport "mannaiah/module/products/port/tag"
)

var (
	// ErrNilService is returned when service dependencies are nil.
	ErrNilService = errors.New("tags service must not be nil")
)

// Authorizer defines authentication and authorization behavior required by tag endpoints.
type Authorizer interface {
	// Require authenticates and authorizes requests using required permissions.
	Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
	// IsUnauthorized reports authentication errors.
	IsUnauthorized(err error) bool
	// IsForbidden reports authorization errors.
	IsForbidden(err error) bool
}

// Handler defines HTTP route handlers for tags and correlations.
type Handler struct {
	// service defines tag use-case dependencies.
	service tagapplication.Service
	// authorizer defines optional endpoint auth dependencies.
	authorizer Authorizer
}

// createCorrelationRequest defines request payload for correlation creation.
type createCorrelationRequest struct {
	SourceTag   string  `json:"sourceTag"`
	TargetTag   string  `json:"targetTag"`
	Probability float64 `json:"probability"`
	Notes       string  `json:"notes"`
}

// updateCorrelationRequest defines request payload for correlation updates.
type updateCorrelationRequest struct {
	Probability *float64 `json:"probability"`
	Notes       *string  `json:"notes"`
}

// deleteResponse defines delete response payload.
type deleteResponse struct {
	Status string `json:"status"`
}

// NewHandler creates tag HTTP handlers.
func NewHandler(service tagapplication.Service, authorizers ...Authorizer) (*Handler, error) {
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

// RegisterRoutes registers tag CRUD and correlation endpoints.
// Correlation routes are registered before :name to avoid path conflicts.
func (h *Handler) RegisterRoutes(router corehttp.Router) {
	router.Get("/tags/correlations", h.protect("marketing:manage", h.listCorrelations))
	router.Get("/tags/correlations/source/:tag", h.protect("marketing:manage", h.listCorrelationsBySource))
	router.Post("/tags/correlations", h.protect("marketing:manage", h.createCorrelation))
	router.Patch("/tags/correlations/:id", h.protect("marketing:manage", h.updateCorrelation))
	router.Delete("/tags/correlations/:id", h.protect("marketing:manage", h.removeCorrelation))
	router.Get("/tags", h.protect("product:tags", h.listTags))
	router.Delete("/tags/:name", h.protect("marketing:manage", h.removeTag))
}

// listTags handles tag listing.
func (h *Handler) listTags(ctx corehttp.Context) error {
	tags, err := h.service.List(ctx.Context())
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(map[string]any{"data": tags})
}

// removeTag handles tag soft-deletion.
func (h *Handler) removeTag(ctx corehttp.Context) error {
	name := strings.TrimSpace(ctx.Params("name"))
	if err := h.service.SoftDelete(ctx.Context(), name); err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(deleteResponse{Status: "deleted"})
}

// listCorrelations handles correlation listing.
func (h *Handler) listCorrelations(ctx corehttp.Context) error {
	correlations, err := h.service.ListCorrelations(ctx.Context())
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(map[string]any{"data": correlations})
}

// listCorrelationsBySource handles correlation listing by source tag.
func (h *Handler) listCorrelationsBySource(ctx corehttp.Context) error {
	sourceTag := strings.TrimSpace(ctx.Params("tag"))
	correlations, err := h.service.ListCorrelationsBySource(ctx.Context(), sourceTag)
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(map[string]any{"data": correlations})
}

// createCorrelation handles correlation creation.
func (h *Handler) createCorrelation(ctx corehttp.Context) error {
	var request createCorrelationRequest
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	correlation, err := h.service.CreateCorrelation(ctx.Context(), tagapplication.CreateCorrelationCommand{
		SourceTag:   request.SourceTag,
		TargetTag:   request.TargetTag,
		Probability: request.Probability,
		Notes:       request.Notes,
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(201).JSON(correlation)
}

// updateCorrelation handles correlation updates.
func (h *Handler) updateCorrelation(ctx corehttp.Context) error {
	id, ok := parseUintParam(ctx, "id")
	if !ok {
		return corehttp.NewAppError(400, "invalid_correlation_id", errors.New("correlation id must be a positive integer"))
	}

	var request updateCorrelationRequest
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	cmd := tagapplication.UpdateCorrelationCommand{}
	if request.Probability != nil {
		cmd.Probability = request.Probability
		cmd.HasProbability = true
	}
	if request.Notes != nil {
		cmd.Notes = request.Notes
		cmd.HasNotes = true
	}

	updated, err := h.service.UpdateCorrelation(ctx.Context(), id, cmd)
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(updated)
}

// removeCorrelation handles correlation deletion.
func (h *Handler) removeCorrelation(ctx corehttp.Context) error {
	id, ok := parseUintParam(ctx, "id")
	if !ok {
		return corehttp.NewAppError(400, "invalid_correlation_id", errors.New("correlation id must be a positive integer"))
	}

	if err := h.service.DeleteCorrelation(ctx.Context(), id); err != nil {
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

// mapError maps domain/application/repository/auth errors to HTTP-layer app errors.
func (h *Handler) mapError(err error) error {
	if err == nil {
		return nil
	}

	if h != nil && h.authorizer != nil {
		if h.authorizer.IsUnauthorized(err) {
			return corehttp.NewAppError(401, "unauthorized", err)
		}
		if h.authorizer.IsForbidden(err) {
			return corehttp.NewAppError(403, "forbidden", err)
		}
	}

	if errors.Is(err, tagport.ErrNotFound) {
		return corehttp.NewAppError(404, "tag_not_found", err)
	}
	if errors.Is(err, tagport.ErrCorrelationNotFound) {
		return corehttp.NewAppError(404, "correlation_not_found", err)
	}
	if errors.Is(err, tagport.ErrDuplicateCorrelation) {
		return corehttp.NewAppError(409, "correlation_pair_conflict", err)
	}
	if errors.Is(err, tagapplication.ErrInvalidTagName) ||
		errors.Is(err, tagapplication.ErrInvalidSourceTag) ||
		errors.Is(err, tagapplication.ErrInvalidTargetTag) ||
		errors.Is(err, tagapplication.ErrProbabilityRange) ||
		errors.Is(err, tagapplication.ErrSelfCorrelation) {
		return corehttp.NewAppError(400, "invalid_tag_request", err)
	}

	return corehttp.NewAppError(500, "internal_server_error", err)
}

// parseUintParam parses a uint URL path parameter by key.
func parseUintParam(ctx corehttp.Context, key string) (uint, bool) {
	raw := strings.TrimSpace(ctx.Params(key))
	if raw == "" {
		return 0, false
	}

	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, false
	}

	return uint(n), true
}
