package http

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	corehttp "mannaiah/module/core/http"
	ordersapplication "mannaiah/module/orders/application"
	ordersdomain "mannaiah/module/orders/domain"
	ordersport "mannaiah/module/orders/port"
)

var (
	// ErrNilService is returned when a nil service dependency is provided.
	ErrNilService = errors.New("orders service must not be nil")
	// ErrInvalidQuery is returned when query parameters are invalid.
	ErrInvalidQuery = errors.New("invalid query parameters")
)

// Authorizer defines authentication and authorization behavior required by order endpoints.
type Authorizer interface {
	// Require authenticates and authorizes requests using required permissions.
	Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
	// IsUnauthorized reports authentication errors.
	IsUnauthorized(err error) bool
	// IsForbidden reports authorization errors.
	IsForbidden(err error) bool
}

// Handler defines HTTP route handlers for orders.
type Handler struct {
	// service defines order use-case dependencies.
	service ordersapplication.Service
	// authorizer defines optional auth dependencies for protected endpoints.
	authorizer Authorizer
}

// createItemRequest defines request payload for order items.
type createItemRequest struct {
	// SKU defines order item SKU values.
	SKU string `json:"sku"`
	// AlternateName defines fallback item name values.
	AlternateName string `json:"alternateName"`
	// Quantity defines ordered quantity values.
	Quantity int `json:"quantity"`
	// Value defines item monetary value values.
	Value float64 `json:"value"`
}

// shippingAddressRequest defines request payload for shipping-address values.
type shippingAddressRequest struct {
	// Address defines address line 1 values.
	Address string `json:"address"`
	// Address2 defines address line 2 values.
	Address2 string `json:"address2"`
	// Phone defines shipping phone values.
	Phone string `json:"phone"`
	// CityCode defines shipping city-code values.
	CityCode string `json:"cityCode"`
}

// shippingChargeRequest defines request payload for shipping charge values.
type shippingChargeRequest struct {
	// MethodID defines shipping method identifier values.
	MethodID string `json:"methodId"`
	// MethodTitle defines shipping method display title values.
	MethodTitle string `json:"methodTitle"`
	// Price defines shipping price values.
	Price float64 `json:"price"`
}

// createRequest defines request payload for order creation.
type createRequest struct {
	// Identifier defines external order identifiers.
	Identifier string `json:"identifier"`
	// Realm defines order realm values.
	Realm string `json:"realm"`
	// ContactID defines customer identifiers.
	ContactID string `json:"contactId"`
	// Items defines ordered item payload values.
	Items []createItemRequest `json:"items"`
	// InitialStatus defines optional initial status values.
	InitialStatus *ordersdomain.Status `json:"initialStatus"`
	// Author defines status author values.
	Author string `json:"author"`
	// Description defines status description values.
	Description string `json:"description"`
	// ShippingAddress defines optional explicit shipping-address values.
	ShippingAddress *shippingAddressRequest `json:"shippingAddress"`
	// ShippingCharges defines shipping charge values.
	ShippingCharges []shippingChargeRequest `json:"shippingCharges"`
	// Metadata defines order metadata values.
	Metadata map[string]string `json:"metadata"`
}

// updateStatusRequest defines request payload for status updates.
type updateStatusRequest struct {
	// Status defines next status values.
	Status ordersdomain.Status `json:"status"`
	// Author defines status author values.
	Author string `json:"author"`
	// Description defines status description values.
	Description string `json:"description"`
	// NoteOwner defines optional note owner values.
	NoteOwner string `json:"noteOwner"`
	// Note defines optional note values.
	Note string `json:"note"`
}

// listMeta defines list response pagination metadata.
type listMeta struct {
	// Page defines current page numbers.
	Page int `json:"page"`
	// Total defines filtered total values.
	Total int64 `json:"total"`
	// Limit defines page-size values.
	Limit int `json:"limit"`
	// TotalPages defines total-page values.
	TotalPages int `json:"totalPages"`
}

// listResponse defines order list response payload.
type listResponse struct {
	// Data defines paged order rows.
	Data []ordersdomain.Order `json:"data"`
	// Meta defines pagination metadata.
	Meta listMeta `json:"meta"`
}

// NewHandler creates an order HTTP handler set.
func NewHandler(service ordersapplication.Service, authorizers ...Authorizer) (*Handler, error) {
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

// RegisterRoutes registers order endpoints.
func (h *Handler) RegisterRoutes(router corehttp.Router) {
	router.Post("/orders", h.protect("orders:manage", h.create))
	router.Get("/orders", h.protect("orders:read", h.findAll))
	router.Get("/orders/:id", h.protect("orders:read", h.findOne))
	router.Patch("/orders/:id/status", h.protect("orders:manage", h.updateStatus))
}

// create handles order creation endpoints.
func (h *Handler) create(ctx corehttp.Context) error {
	var request createRequest
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	command := ordersapplication.CreateCommand{
		Identifier:      request.Identifier,
		Realm:           request.Realm,
		ContactID:       request.ContactID,
		InitialStatus:   request.InitialStatus,
		Author:          request.Author,
		Description:     request.Description,
		Items:           mapCreateItems(request.Items),
		ShippingCharges: mapShippingCharges(request.ShippingCharges),
		Metadata:        request.Metadata,
	}
	if request.ShippingAddress != nil {
		command.ShippingAddress = &ordersapplication.ShippingAddressCommand{
			Address:  request.ShippingAddress.Address,
			Address2: request.ShippingAddress.Address2,
			Phone:    request.ShippingAddress.Phone,
			CityCode: request.ShippingAddress.CityCode,
		}
	}

	entity, err := h.service.Create(ctx.Context(), command)
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(201).JSON(entity)
}

// findAll handles paginated order listing endpoints.
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

// findOne handles order-by-id retrieval endpoints.
func (h *Handler) findOne(ctx corehttp.Context) error {
	entity, err := h.service.Get(ctx.Context(), ctx.Params("id"))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(entity)
}

// updateStatus handles order status update endpoints.
func (h *Handler) updateStatus(ctx corehttp.Context) error {
	var request updateStatusRequest
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	entity, err := h.service.UpdateStatus(ctx.Context(), ctx.Params("id"), ordersapplication.UpdateStatusCommand{
		Status:      request.Status,
		Author:      request.Author,
		Description: request.Description,
		NoteOwner:   request.NoteOwner,
		Note:        request.Note,
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(entity)
}

// mapCreateItems maps request item payloads to application command values.
func mapCreateItems(items []createItemRequest) []ordersapplication.CreateItemCommand {
	result := make([]ordersapplication.CreateItemCommand, 0, len(items))
	for _, item := range items {
		result = append(result, ordersapplication.CreateItemCommand{
			SKU:           item.SKU,
			AlternateName: item.AlternateName,
			Quantity:      item.Quantity,
			Value:         item.Value,
		})
	}

	return result
}

// mapShippingCharges maps shipping-charge request payloads to application command values.
func mapShippingCharges(values []shippingChargeRequest) []ordersapplication.ShippingChargeCommand {
	result := make([]ordersapplication.ShippingChargeCommand, 0, len(values))
	for _, value := range values {
		result = append(result, ordersapplication.ShippingChargeCommand{
			MethodID:    value.MethodID,
			MethodTitle: value.MethodTitle,
			Price:       value.Price,
		})
	}

	return result
}

// parseListQuery parses list query params into application query structures.
func parseListQuery(ctx corehttp.Context) (ordersapplication.ListQuery, error) {
	page, err := parsePositiveInt(ctx.Query("page", "1"))
	if err != nil {
		return ordersapplication.ListQuery{}, fmt.Errorf("page: %w", err)
	}
	limit, err := parsePositiveInt(ctx.Query("limit", "10"))
	if err != nil {
		return ordersapplication.ListQuery{}, fmt.Errorf("limit: %w", err)
	}

	return ordersapplication.ListQuery{
		Page:       page,
		Limit:      limit,
		Realm:      strings.TrimSpace(ctx.Query("realm", "")),
		ContactID:  strings.TrimSpace(ctx.Query("contactId", "")),
		Identifier: strings.TrimSpace(ctx.Query("identifier", "")),
		Status:     ordersdomain.Status(strings.ToUpper(strings.TrimSpace(ctx.Query("status", "")))),
	}, nil
}

// parsePositiveInt parses positive integer query values.
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
	if errors.Is(err, ordersport.ErrNotFound) {
		return corehttp.NewAppError(404, "order_not_found", err)
	}
	if errors.Is(err, ordersapplication.ErrInvalidID) {
		return corehttp.NewAppError(400, "invalid_order_id", err)
	}
	if errors.Is(err, ordersport.ErrDuplicateIdentifier) {
		return corehttp.NewAppError(409, "order_identifier_conflict", err)
	}
	if errors.Is(err, ordersport.ErrCustomerNotFound) {
		return corehttp.NewAppError(404, "order_customer_not_found", err)
	}
	if errors.Is(err, ErrInvalidQuery) ||
		errors.Is(err, ordersapplication.ErrStatusAuthorRequired) ||
		errors.Is(err, ordersdomain.ErrIdentifierRequired) ||
		errors.Is(err, ordersdomain.ErrRealmRequired) ||
		errors.Is(err, ordersdomain.ErrContactIDRequired) ||
		errors.Is(err, ordersdomain.ErrItemsRequired) ||
		errors.Is(err, ordersdomain.ErrItemIdentifierRequired) ||
		errors.Is(err, ordersdomain.ErrItemQuantityInvalid) ||
		errors.Is(err, ordersdomain.ErrStatusInvalid) ||
		errors.Is(err, ordersdomain.ErrStatusAuthorRequired) {
		return corehttp.NewAppError(400, "invalid_order", err)
	}

	return corehttp.NewAppError(500, "internal_server_error", err)
}
