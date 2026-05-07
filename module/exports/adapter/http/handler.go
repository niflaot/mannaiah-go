package http

import (
	"context"
	"errors"
	"strconv"
	"strings"

	corehttp "mannaiah/module/core/http"
	"mannaiah/module/exports/application"
	"mannaiah/module/exports/domain"
	"mannaiah/module/exports/port"
)

var (
	// ErrNilService is returned when export services are nil.
	ErrNilService = errors.New("exports service must not be nil")
)

// Authorizer defines authentication and authorization behavior required by export endpoints.
type Authorizer interface {
	// Require authenticates and authorizes requests using required permissions.
	Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
	// IsUnauthorized reports authentication errors.
	IsUnauthorized(err error) bool
	// IsForbidden reports authorization errors.
	IsForbidden(err error) bool
}

// Service defines export use cases required by HTTP handlers.
type Service interface {
	// GenerateContacts creates a contact CSV report.
	GenerateContacts(ctx context.Context) (*domain.Report, error)
	// GenerateOrders creates an order CSV report.
	GenerateOrders(ctx context.Context) (*domain.Report, error)
	// GetReport retrieves one report by id.
	GetReport(ctx context.Context, id string) (*domain.Report, error)
	// ListReports returns paginated reports.
	ListReports(ctx context.Context, query port.ListQuery) (*application.ListResult, error)
	// SearchReports returns paginated reports using filter criteria.
	SearchReports(ctx context.Context, query port.ListQuery) (*application.ListResult, error)
}

// Handler defines export HTTP route handlers.
type Handler struct {
	// service defines export use-case dependencies.
	service Service
	// authorizer defines optional auth dependencies.
	authorizer Authorizer
}

// NewHandler creates export HTTP handlers.
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

// RegisterRoutes registers export routes.
func (h *Handler) RegisterRoutes(router corehttp.Router) {
	router.Post("/exports/contacts", h.protect("marketing:manage", h.generateContacts))
	router.Post("/exports/orders", h.protect("marketing:manage", h.generateOrders))
	router.Post("/export/orders", h.protect("marketing:manage", h.generateOrders))
	router.Get("/exports/reports", h.protect("marketing:manage", h.listReports))
	router.Get("/exports/reports/:id", h.protect("marketing:manage", h.getReport))
	router.Get("/exports/search", h.protect("marketing:manage", h.searchReports))
}

// generateContacts handles contact export generation requests.
func (h *Handler) generateContacts(ctx corehttp.Context) error {
	report, err := h.service.GenerateContacts(ctx.Context())
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(201).JSON(report)
}

// generateOrders handles order export generation requests.
func (h *Handler) generateOrders(ctx corehttp.Context) error {
	report, err := h.service.GenerateOrders(ctx.Context())
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(201).JSON(report)
}

// getReport handles report registry lookup requests.
func (h *Handler) getReport(ctx corehttp.Context) error {
	report, err := h.service.GetReport(ctx.Context(), strings.TrimSpace(ctx.Params("id")))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(report)
}

// listReports handles report registry list requests.
func (h *Handler) listReports(ctx corehttp.Context) error {
	query, err := parseListQuery(ctx)
	if err != nil {
		return corehttp.NewAppError(400, "invalid_query", err)
	}
	result, serviceErr := h.service.ListReports(ctx.Context(), query)
	if serviceErr != nil {
		return h.mapError(serviceErr)
	}

	return ctx.Status(200).JSON(result)
}

// searchReports handles report registry search requests.
func (h *Handler) searchReports(ctx corehttp.Context) error {
	query, err := parseListQuery(ctx)
	if err != nil {
		return corehttp.NewAppError(400, "invalid_query", err)
	}
	result, serviceErr := h.service.SearchReports(ctx.Context(), query)
	if serviceErr != nil {
		return h.mapError(serviceErr)
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
	if errors.Is(err, domain.ErrReportNotFound) {
		return corehttp.NewAppError(404, "export_report_not_found", err)
	}
	if errors.Is(err, domain.ErrInvalidReportID) || errors.Is(err, domain.ErrInvalidReportType) {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	return corehttp.NewAppError(500, "internal_server_error", err)
}

// parseListQuery parses report registry query values.
func parseListQuery(ctx corehttp.Context) (port.ListQuery, error) {
	page, err := parsePositiveInt(ctx.Query("page", "1"))
	if err != nil {
		return port.ListQuery{}, err
	}
	limit, err := parsePositiveInt(ctx.Query("limit", "50"))
	if err != nil {
		return port.ListQuery{}, err
	}

	return port.ListQuery{
		Type:  domain.ReportType(strings.TrimSpace(ctx.Query("type"))),
		Page:  page,
		Limit: limit,
	}, nil
}

// parsePositiveInt parses positive integer query values.
func parsePositiveInt(value string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return 0, errors.New("value must be a positive integer")
	}

	return parsed, nil
}
