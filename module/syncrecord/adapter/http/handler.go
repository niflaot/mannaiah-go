package http

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	corehttp "mannaiah/module/core/http"
	"mannaiah/module/syncrecord/application"
	"mannaiah/module/syncrecord/domain"
	"mannaiah/module/syncrecord/port"
)

var (
	// ErrNilService is returned when service dependencies are nil.
	ErrNilService = errors.New("sync record service must not be nil")
)

// Authorizer defines authentication and authorization behavior required by sync record endpoints.
type Authorizer interface {
	// Require authenticates and authorizes requests using required permissions.
	Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
	// IsUnauthorized reports authentication errors.
	IsUnauthorized(err error) bool
	// IsForbidden reports authorization errors.
	IsForbidden(err error) bool
}

// Service defines sync record use-case behavior required by HTTP handlers.
type Service interface {
	// GetRun retrieves one run by id.
	GetRun(ctx context.Context, runID string) (*domain.SyncRun, error)
	// ListRuns returns paged run rows using filter query values.
	ListRuns(ctx context.Context, query port.ListQuery) (*application.ListResult, error)
	// StatsSince returns aggregate stats since one timestamp.
	StatsSince(ctx context.Context, since time.Time) (*domain.RunStats, error)
}

// Handler defines HTTP route handlers for sync record endpoints.
type Handler struct {
	// service defines sync record use-case dependencies.
	service Service
	// authorizer defines optional auth dependencies.
	authorizer Authorizer
}

// NewHandler creates sync record HTTP handlers.
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

// SetAuthorizer configures endpoint authentication and authorization dependencies.
func (h *Handler) SetAuthorizer(authorizer Authorizer) {
	if h == nil {
		return
	}

	h.authorizer = authorizer
}

// RegisterRoutes registers sync record routes.
func (h *Handler) RegisterRoutes(router corehttp.Router) {
	router.Get("/syncrecord/runs", h.protect("marketing:manage", h.listRuns))
	router.Get("/syncrecord/runs/:id", h.protect("marketing:manage", h.getRun))
	router.Get("/syncrecord/stats", h.protect("marketing:manage", h.stats))
}

// listRuns handles paged sync run listing requests.
func (h *Handler) listRuns(ctx corehttp.Context) error {
	query, err := parseListQuery(ctx)
	if err != nil {
		return corehttp.NewAppError(400, "invalid_query", err)
	}

	result, serviceErr := h.service.ListRuns(ctx.Context(), query)
	if serviceErr != nil {
		return h.mapError(serviceErr)
	}

	return ctx.Status(200).JSON(result)
}

// getRun handles sync run by-id requests.
func (h *Handler) getRun(ctx corehttp.Context) error {
	run, err := h.service.GetRun(ctx.Context(), strings.TrimSpace(ctx.Params("id")))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(run)
}

// stats handles sync run aggregate stats requests.
func (h *Handler) stats(ctx corehttp.Context) error {
	windowHours := 24
	if rawWindow := strings.TrimSpace(ctx.Query("windowHours", "24")); rawWindow != "" {
		value, err := strconv.Atoi(rawWindow)
		if err != nil || value <= 0 {
			return corehttp.NewAppError(400, "invalid_query", errors.New("windowHours must be a positive integer"))
		}
		windowHours = value
	}

	since := time.Now().UTC().Add(-time.Duration(windowHours) * time.Hour)
	result, err := h.service.StatsSince(ctx.Context(), since)
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

// mapError maps service and auth errors to HTTP-layer app errors.
func (h *Handler) mapError(err error) error {
	if h != nil && h.authorizer != nil {
		if h.authorizer.IsUnauthorized(err) {
			return corehttp.NewAppError(401, "unauthorized", err)
		}
		if h.authorizer.IsForbidden(err) {
			return corehttp.NewAppError(403, "forbidden", err)
		}
	}
	if errors.Is(err, domain.ErrRunNotFound) {
		return corehttp.NewAppError(404, "sync_run_not_found", err)
	}
	if errors.Is(err, domain.ErrInvalidRunID) || errors.Is(err, domain.ErrInvalidKind) || errors.Is(err, domain.ErrInvalidTrigger) {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	return corehttp.NewAppError(500, "internal_server_error", err)
}

// parseListQuery parses query params into list query values.
func parseListQuery(ctx corehttp.Context) (port.ListQuery, error) {
	page, err := parsePositiveInt(ctx.Query("page", "1"))
	if err != nil {
		return port.ListQuery{}, err
	}
	limit, err := parsePositiveInt(ctx.Query("limit", "50"))
	if err != nil {
		return port.ListQuery{}, err
	}

	query := port.ListQuery{
		Kind:    strings.TrimSpace(ctx.Query("kind")),
		Trigger: strings.TrimSpace(ctx.Query("trigger")),
		Status:  strings.TrimSpace(ctx.Query("status")),
		Page:    page,
		Limit:   limit,
	}
	if startedAfter := strings.TrimSpace(ctx.Query("startedAfter")); startedAfter != "" {
		value, parseErr := time.Parse(time.RFC3339, startedAfter)
		if parseErr != nil {
			return port.ListQuery{}, parseErr
		}
		query.StartedAfter = &value
	}
	if startedBefore := strings.TrimSpace(ctx.Query("startedBefore")); startedBefore != "" {
		value, parseErr := time.Parse(time.RFC3339, startedBefore)
		if parseErr != nil {
			return port.ListQuery{}, parseErr
		}
		query.StartedBefore = &value
	}

	return query, nil
}

// parsePositiveInt parses positive integer query values.
func parsePositiveInt(value string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return 0, errors.New("value must be a positive integer")
	}

	return parsed, nil
}
