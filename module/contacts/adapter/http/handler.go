package http

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"mannaiah/module/contacts/application"
	"mannaiah/module/contacts/domain"
	"mannaiah/module/contacts/port"
	corehttp "mannaiah/module/core/http"
)

var (
	// ErrNilService is returned when a nil service dependency is provided.
	ErrNilService = errors.New("contacts service must not be nil")
	// ErrInvalidQuery is returned when query parameters are invalid.
	ErrInvalidQuery = errors.New("invalid query parameters")
)

// Authorizer defines authentication and authorization behavior required by contact endpoints.
type Authorizer interface {
	// Require authenticates and authorizes requests using required permissions.
	Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
	// IsUnauthorized reports authentication errors.
	IsUnauthorized(err error) bool
	// IsForbidden reports authorization errors.
	IsForbidden(err error) bool
}

// Handler defines HTTP route handlers for contacts.
type Handler struct {
	// service defines contact use-case dependency.
	service application.Service
	// authorizer defines optional auth dependency for protected endpoints.
	authorizer Authorizer
}

// createRequest defines request payload for contact creation.
type createRequest struct {
	// DocumentType defines document category.
	DocumentType domain.DocumentType `json:"documentType"`
	// DocumentNumber defines document number.
	DocumentNumber string `json:"documentNumber"`
	// LegalName defines legal names.
	LegalName string `json:"legalName"`
	// FirstName defines first names.
	FirstName string `json:"firstName"`
	// LastName defines last names.
	LastName string `json:"lastName"`
	// Email defines email values.
	Email string `json:"email"`
	// Phone defines phone values.
	Phone string `json:"phone"`
	// Address defines address values.
	Address string `json:"address"`
	// AddressExtra defines extra address values.
	AddressExtra string `json:"addressExtra"`
	// CityCode defines city code values.
	CityCode string `json:"cityCode"`
}

// updateRequest defines request payload for contact updates.
type updateRequest struct {
	// DocumentType defines optional document category updates.
	DocumentType *domain.DocumentType `json:"documentType"`
	// DocumentNumber defines optional document number updates.
	DocumentNumber *string `json:"documentNumber"`
	// LegalName defines optional legal name updates.
	LegalName *string `json:"legalName"`
	// FirstName defines optional first name updates.
	FirstName *string `json:"firstName"`
	// LastName defines optional last name updates.
	LastName *string `json:"lastName"`
	// Email defines optional email updates.
	Email *string `json:"email"`
	// Phone defines optional phone updates.
	Phone *string `json:"phone"`
	// Address defines optional address updates.
	Address *string `json:"address"`
	// AddressExtra defines optional address extra updates.
	AddressExtra *string `json:"addressExtra"`
	// CityCode defines optional city code updates.
	CityCode *string `json:"cityCode"`
}

// listMeta defines list response pagination metadata.
type listMeta struct {
	// Page defines current page number.
	Page int `json:"page"`
	// Total defines total records after filtering.
	Total int64 `json:"total"`
	// Limit defines page size.
	Limit int `json:"limit"`
	// TotalPages defines total pages.
	TotalPages int `json:"totalPages"`
}

// listResponse defines contact list response payload.
type listResponse struct {
	// Data defines paged contact rows.
	Data []domain.Contact `json:"data"`
	// Meta defines pagination metadata.
	Meta listMeta `json:"meta"`
}

// NewHandler creates a contact HTTP handler set.
func NewHandler(service application.Service, authorizers ...Authorizer) (*Handler, error) {
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

// RegisterRoutes registers contact CRUD endpoints.
func (h *Handler) RegisterRoutes(router corehttp.Router) {
	router.Post("/contacts", h.protect("contacts:create", h.create))
	router.Get("/contacts", h.protect("contacts:read", h.findAll))
	router.Get("/contacts/:id", h.protect("contacts:read", h.findOne))
	router.Patch("/contacts/:id", h.protect("contacts:update", h.update))
	router.Delete("/contacts/:id", h.protect("contacts:delete", h.remove))
}

// create handles contact creation endpoints.
func (h *Handler) create(ctx corehttp.Context) error {
	var request createRequest
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	contact, err := h.service.Create(ctx.Context(), application.CreateCommand{
		DocumentType:   request.DocumentType,
		DocumentNumber: request.DocumentNumber,
		LegalName:      request.LegalName,
		FirstName:      request.FirstName,
		LastName:       request.LastName,
		Email:          request.Email,
		Phone:          request.Phone,
		Address:        request.Address,
		AddressExtra:   request.AddressExtra,
		CityCode:       request.CityCode,
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(201).JSON(contact)
}

// findAll handles paginated contact listing endpoints.
func (h *Handler) findAll(ctx corehttp.Context) error {
	query, err := parseListQuery(ctx)
	if err != nil {
		return corehttp.NewAppError(400, "invalid_query", err)
	}

	page, err := h.service.List(ctx.Context(), query)
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(listResponse{
		Data: page.Data,
		Meta: listMeta{
			Page:       page.Page,
			Total:      page.Total,
			Limit:      page.Limit,
			TotalPages: page.TotalPages,
		},
	})
}

// findOne handles contact-by-id retrieval endpoints.
func (h *Handler) findOne(ctx corehttp.Context) error {
	contact, err := h.service.Get(ctx.Context(), ctx.Params("id"))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(contact)
}

// update handles contact update endpoints.
func (h *Handler) update(ctx corehttp.Context) error {
	var request updateRequest
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	contact, err := h.service.Update(ctx.Context(), ctx.Params("id"), application.UpdateCommand{
		DocumentType:   request.DocumentType,
		DocumentNumber: request.DocumentNumber,
		LegalName:      request.LegalName,
		FirstName:      request.FirstName,
		LastName:       request.LastName,
		Email:          request.Email,
		Phone:          request.Phone,
		Address:        request.Address,
		AddressExtra:   request.AddressExtra,
		CityCode:       request.CityCode,
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(contact)
}

// remove handles contact delete endpoints.
func (h *Handler) remove(ctx corehttp.Context) error {
	if err := h.service.Delete(ctx.Context(), ctx.Params("id")); err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(map[string]string{"status": "deleted"})
}

// parseListQuery parses list query params into application query structures.
func parseListQuery(ctx corehttp.Context) (port.ListQuery, error) {
	page, err := parsePositiveInt(ctx.Query("page", "1"))
	if err != nil {
		return port.ListQuery{}, fmt.Errorf("page: %w", err)
	}
	limit, err := parsePositiveInt(ctx.Query("limit", "10"))
	if err != nil {
		return port.ListQuery{}, fmt.Errorf("limit: %w", err)
	}

	return port.ListQuery{
		Page:       page,
		Limit:      limit,
		OrderBy:    strings.TrimSpace(ctx.Query("orderBy", "")),
		OrderDir:   strings.TrimSpace(ctx.Query("orderDir", "")),
		Email:      strings.TrimSpace(ctx.Query("email", "")),
		ExcludeIDs: parseExcludedIDs(ctx.Query("excludeIds", "")),
	}, nil
}

// parsePositiveInt parses positive integer values.
func parsePositiveInt(raw string) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, ErrInvalidQuery
	}

	value, err := strconv.Atoi(trimmed)
	if err != nil || value <= 0 {
		return 0, ErrInvalidQuery
	}

	return value, nil
}

// parseExcludedIDs parses comma-separated exclusion IDs.
func parseExcludedIDs(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}

	parts := strings.Split(trimmed, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			result = append(result, value)
		}
	}

	return result
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

// mapError maps application/domain/repository/auth errors to HTTP-layer app errors.
func (h *Handler) mapError(err error) error {
	if h != nil && h.authorizer != nil {
		if h.authorizer.IsUnauthorized(err) {
			return corehttp.NewAppError(401, "unauthorized", err)
		}
		if h.authorizer.IsForbidden(err) {
			return corehttp.NewAppError(403, "forbidden", err)
		}
	}
	if errors.Is(err, port.ErrNotFound) {
		return corehttp.NewAppError(404, "contact_not_found", err)
	}
	if errors.Is(err, application.ErrInvalidID) {
		return corehttp.NewAppError(400, "invalid_contact_id", err)
	}
	if errors.Is(err, domain.ErrEmailRequired) || errors.Is(err, domain.ErrInvalidNameCombination) || errors.Is(err, domain.ErrIncompletePersonalName) {
		return corehttp.NewAppError(400, "invalid_contact", err)
	}
	if errors.Is(err, ErrInvalidQuery) {
		return corehttp.NewAppError(400, "invalid_query", err)
	}

	return corehttp.NewAppError(500, "internal_server_error", err)
}
