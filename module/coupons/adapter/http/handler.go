// Package http defines coupon HTTP route handlers.
package http

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	corehttp "mannaiah/module/core/http"
	couponservice "mannaiah/module/coupons/application/coupon/service"
	"mannaiah/module/coupons/domain"
	"mannaiah/module/coupons/port"
)

var (
	// ErrNilService is returned when a nil service dependency is provided.
	ErrNilService = errors.New("coupon service must not be nil")
)

// Authorizer defines authentication and authorization behavior for coupon endpoints.
type Authorizer interface {
	// Require authenticates and authorizes requests using required permissions.
	Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
	// IsUnauthorized reports authentication errors.
	IsUnauthorized(err error) bool
	// IsForbidden reports authorization errors.
	IsForbidden(err error) bool
}

// Handler defines HTTP route handlers for the coupons feature.
type Handler struct {
	// service defines coupon use-case dependencies.
	service *couponservice.Service
	// authorizer defines optional auth dependencies for protected endpoints.
	authorizer Authorizer
}

// NewHandler creates coupon HTTP handlers.
func NewHandler(service *couponservice.Service) (*Handler, error) {
	if service == nil {
		return nil, ErrNilService
	}
	return &Handler{service: service}, nil
}

// SetAuthorizer configures endpoint authentication dependencies.
func (h *Handler) SetAuthorizer(authorizer Authorizer) {
	if h == nil {
		return
	}
	h.authorizer = authorizer
}

// RegisterRoutes mounts all coupon routes onto the provided router.
func (h *Handler) RegisterRoutes(router corehttp.Router) {
	router.Post("/coupons", h.protect(h.create, "coupon:manage"))
	router.Get("/coupons", h.protect(h.list, "coupon:view"))
	router.Get("/coupons/:id", h.protect(h.getByID, "coupon:view"))
	router.Put("/coupons/:id", h.protect(h.update, "coupon:manage"))
	router.Delete("/coupons/:id", h.protect(h.delete, "coupon:manage"))
	router.Get("/coupons/code/:code", h.protect(h.getByCode, "coupon:view"))
	router.Post("/coupons/:id/usage", h.protect(h.recordUsage, "coupon:manage"))
	router.Get("/search/coupons", h.protectAny(h.search, "coupon:view", "coupon:manage", "coupon:sync"))
}

// createRequest defines coupon creation request payload values.
type createRequest struct {
	// Code defines optional explicit code values. When empty a random code is generated.
	Code string `json:"code"`
	// Origin defines the coupon source.
	Origin string `json:"origin"`
	// DiscountType defines the discount method ("fixed" or "percentage").
	DiscountType string `json:"discountType"`
	// DiscountAmount defines the discount value.
	DiscountAmount float64 `json:"discountAmount"`
	// MaxUsagesGlobal defines the optional global usage limit.
	MaxUsagesGlobal *int `json:"maxUsagesGlobal,omitempty"`
	// MaxUsagesPerEmail defines the optional per-email usage limit.
	MaxUsagesPerEmail *int `json:"maxUsagesPerEmail,omitempty"`
	// Active defines the initial active state.
	Active bool `json:"active"`
	// ExpiresAt defines the optional expiry timestamp (RFC3339).
	ExpiresAt *string `json:"expiresAt,omitempty"`
	// AssignedEmails defines the optional list of authorized emails.
	AssignedEmails []string `json:"assignedEmails,omitempty"`
	// AssignedContactIDs defines the optional list of authorized contact identifiers.
	AssignedContactIDs []string `json:"assignedContactIds,omitempty"`
	// IncludedProductIDs defines the optional product scope.
	IncludedProductIDs []string `json:"includedProductIds,omitempty"`
	// IncludedCategoryIDs defines the optional category scope.
	IncludedCategoryIDs []string `json:"includedCategoryIds,omitempty"`
	// IncludedTagIDs defines the optional tag scope.
	IncludedTagIDs []string `json:"includedTagIds,omitempty"`
}

// updateRequest defines coupon mutation request payload values.
type updateRequest struct {
	// Origin defines the updated origin value.
	Origin string `json:"origin"`
	// DiscountType defines the updated discount method.
	DiscountType string `json:"discountType"`
	// DiscountAmount defines the updated discount value.
	DiscountAmount float64 `json:"discountAmount"`
	// MaxUsagesGlobal defines the updated global usage limit. Nil clears the limit.
	MaxUsagesGlobal *int `json:"maxUsagesGlobal,omitempty"`
	// MaxUsagesPerEmail defines the updated per-email usage limit. Nil clears the limit.
	MaxUsagesPerEmail *int `json:"maxUsagesPerEmail,omitempty"`
	// Active defines the updated active state.
	Active bool `json:"active"`
	// ExpiresAt defines the updated expiry timestamp (RFC3339). Nil clears the expiry.
	ExpiresAt *string `json:"expiresAt,omitempty"`
	// AssignedEmails replaces the list of authorized emails.
	AssignedEmails []string `json:"assignedEmails,omitempty"`
	// AssignedContactIDs replaces the list of authorized contact identifiers.
	AssignedContactIDs []string `json:"assignedContactIds,omitempty"`
	// IncludedProductIDs replaces the product scope.
	IncludedProductIDs []string `json:"includedProductIds,omitempty"`
	// IncludedCategoryIDs replaces the category scope.
	IncludedCategoryIDs []string `json:"includedCategoryIds,omitempty"`
	// IncludedTagIDs replaces the tag scope.
	IncludedTagIDs []string `json:"includedTagIds,omitempty"`
}

// recordUsageRequest defines coupon usage recording request payload values.
type recordUsageRequest struct {
	// OrderID defines the order where the coupon was applied.
	OrderID string `json:"orderId"`
	// Email defines the email of the redeemer.
	Email string `json:"email"`
}

// listResponse defines the coupon list response payload.
type listResponse struct {
	// Items defines the returned coupon rows.
	Items []domain.Coupon `json:"items"`
	// Total defines the unfiltered count.
	Total int64 `json:"total"`
}

// create handles POST /coupons.
func (h *Handler) create(ctx corehttp.Context) error {
	var req createRequest
	if err := ctx.BodyParser(&req); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	expiresAt, err := parseOptionalTime(req.ExpiresAt)
	if err != nil {
		return corehttp.NewAppError(400, "invalid_expires_at", err)
	}

	coupon, err := h.service.Create(ctx.Context(), couponservice.CreateCommand{
		Code:                strings.TrimSpace(req.Code),
		Origin:              strings.TrimSpace(req.Origin),
		DiscountType:        domain.DiscountType(strings.TrimSpace(req.DiscountType)),
		DiscountAmount:      req.DiscountAmount,
		MaxUsagesGlobal:     req.MaxUsagesGlobal,
		MaxUsagesPerEmail:   req.MaxUsagesPerEmail,
		Active:              req.Active,
		ExpiresAt:           expiresAt,
		AssignedEmails:      req.AssignedEmails,
		AssignedContactIDs:  req.AssignedContactIDs,
		IncludedProductIDs:  req.IncludedProductIDs,
		IncludedCategoryIDs: req.IncludedCategoryIDs,
		IncludedTagIDs:      req.IncludedTagIDs,
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(201).JSON(coupon)
}

// list handles GET /coupons.
func (h *Handler) list(ctx corehttp.Context) error {
	query := port.ListQuery{
		Origin: strings.TrimSpace(ctx.Query("origin")),
		Code:   strings.TrimSpace(ctx.Query("code")),
		Limit:  parseIntQuery(ctx, "limit", 50),
		Offset: parseIntQuery(ctx, "offset", 0),
	}
	if activeStr := strings.TrimSpace(ctx.Query("active")); activeStr != "" {
		b, err := strconv.ParseBool(activeStr)
		if err == nil {
			query.Active = &b
		}
	}

	coupons, total, err := h.service.List(ctx.Context(), query)
	if err != nil {
		return corehttp.NewAppError(500, "internal_server_error", err)
	}

	return ctx.Status(200).JSON(listResponse{Items: coupons, Total: total})
}

// getByID handles GET /coupons/:id.
func (h *Handler) getByID(ctx corehttp.Context) error {
	coupon, err := h.service.GetByID(ctx.Context(), ctx.Params("id"))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(coupon)
}

// getByCode handles GET /coupons/code/:code.
func (h *Handler) getByCode(ctx corehttp.Context) error {
	coupon, err := h.service.GetByCode(ctx.Context(), ctx.Params("code"))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(coupon)
}

// update handles PUT /coupons/:id.
func (h *Handler) update(ctx corehttp.Context) error {
	var req updateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	expiresAt, err := parseOptionalTime(req.ExpiresAt)
	if err != nil {
		return corehttp.NewAppError(400, "invalid_expires_at", err)
	}

	coupon, err := h.service.Update(ctx.Context(), couponservice.UpdateCommand{
		ID:                  ctx.Params("id"),
		Origin:              strings.TrimSpace(req.Origin),
		DiscountType:        domain.DiscountType(strings.TrimSpace(req.DiscountType)),
		DiscountAmount:      req.DiscountAmount,
		MaxUsagesGlobal:     req.MaxUsagesGlobal,
		MaxUsagesPerEmail:   req.MaxUsagesPerEmail,
		Active:              req.Active,
		ExpiresAt:           expiresAt,
		AssignedEmails:      req.AssignedEmails,
		AssignedContactIDs:  req.AssignedContactIDs,
		IncludedProductIDs:  req.IncludedProductIDs,
		IncludedCategoryIDs: req.IncludedCategoryIDs,
		IncludedTagIDs:      req.IncludedTagIDs,
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(coupon)
}

// delete handles DELETE /coupons/:id.
func (h *Handler) delete(ctx corehttp.Context) error {
	if err := h.service.Delete(ctx.Context(), ctx.Params("id")); err != nil {
		return h.mapError(err)
	}

	return ctx.SendStatus(204)
}

// recordUsage handles POST /coupons/:id/usage.
func (h *Handler) recordUsage(ctx corehttp.Context) error {
	var req recordUsageRequest
	if err := ctx.BodyParser(&req); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	if err := h.service.RecordUsage(ctx.Context(), couponservice.RecordUsageCommand{
		CouponID: ctx.Params("id"),
		OrderID:  strings.TrimSpace(req.OrderID),
		Email:    strings.TrimSpace(req.Email),
	}); err != nil {
		return h.mapError(err)
	}

	return ctx.SendStatus(204)
}

// search handles GET /search/coupons.
func (h *Handler) search(ctx corehttp.Context) error {
	term := strings.TrimSpace(ctx.Query("term"))
	if term == "" {
		term = strings.TrimSpace(ctx.Query("search"))
	}

	query := port.SearchQuery{
		Term:         term,
		Email:        strings.ToLower(strings.TrimSpace(ctx.Query("email"))),
		ContactID:    strings.TrimSpace(ctx.Query("contact")),
		Code:         strings.TrimSpace(ctx.Query("code")),
		Origin:       strings.TrimSpace(ctx.Query("origin")),
		DiscountType: strings.TrimSpace(ctx.Query("discountType")),
		Limit:        parseIntQuery(ctx, "limit", 50),
		Offset:       parseIntQuery(ctx, "offset", 0),
	}

	coupons, total, err := h.service.Search(ctx.Context(), query)
	if err != nil {
		return corehttp.NewAppError(500, "internal_server_error", err)
	}

	return ctx.Status(200).JSON(listResponse{Items: coupons, Total: total})
}

// protect wraps endpoint handlers with optional authentication and permission checks.
func (h *Handler) protect(next corehttp.Handler, permissions ...string) corehttp.Handler {
	if h == nil || h.authorizer == nil {
		return next
	}

	return func(ctx corehttp.Context) error {
		err := h.authorizer.Require(ctx.Context(), ctx.GetHeader("Authorization"), permissions...)
		if err != nil {
			return h.mapError(err)
		}

		return next(ctx)
	}
}

// protectAny wraps an endpoint handler requiring the token to hold at least one of the given permissions.
func (h *Handler) protectAny(next corehttp.Handler, permissions ...string) corehttp.Handler {
	if h == nil || h.authorizer == nil {
		return next
	}

	return func(ctx corehttp.Context) error {
		header := ctx.GetHeader("Authorization")
		for _, perm := range permissions {
			if err := h.authorizer.Require(ctx.Context(), header, perm); err == nil {
				return next(ctx)
			}
		}
		return corehttp.NewAppError(403, "forbidden", nil)
	}
}

// mapError maps service-layer errors to HTTP app errors.
func (h *Handler) mapError(err error) error {
	if h != nil && h.authorizer != nil {
		if h.authorizer.IsUnauthorized(err) {
			return corehttp.NewAppError(401, "unauthorized", err)
		}
		if h.authorizer.IsForbidden(err) {
			return corehttp.NewAppError(403, "forbidden", err)
		}
	}
	switch {
	case errors.Is(err, couponservice.ErrCouponNotFound):
		return corehttp.NewAppError(404, "coupon_not_found", err)
	case errors.Is(err, couponservice.ErrCouponCodeConflict):
		return corehttp.NewAppError(409, "coupon_code_conflict", err)
	case errors.Is(err, couponservice.ErrCouponAlreadyUsedOnOrder):
		return corehttp.NewAppError(409, "coupon_already_used_on_order", err)
	case errors.Is(err, couponservice.ErrCouponExhausted):
		return corehttp.NewAppError(422, "coupon_exhausted", err)
	case errors.Is(err, couponservice.ErrCouponExhaustedPerEmail):
		return corehttp.NewAppError(422, "coupon_exhausted_per_email", err)
	case errors.Is(err, couponservice.ErrCouponExpired):
		return corehttp.NewAppError(422, "coupon_expired", err)
	case errors.Is(err, couponservice.ErrCouponInactive):
		return corehttp.NewAppError(422, "coupon_inactive", err)
	case errors.Is(err, domain.ErrDiscountTypeInvalid),
		errors.Is(err, domain.ErrDiscountAmountInvalid),
		errors.Is(err, domain.ErrCodeRequired),
		errors.Is(err, domain.ErrPercentageExceedsMax):
		return corehttp.NewAppError(400, "invalid_coupon", err)
	default:
		return corehttp.NewAppError(500, "internal_server_error", err)
	}
}

// parseOptionalTime parses an optional RFC3339 timestamp string.
func parseOptionalTime(value *string) (*time.Time, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(*value))
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// parseIntQuery parses an integer query parameter with a default fallback.
func parseIntQuery(ctx corehttp.Context, key string, defaultValue int) int {
	raw := strings.TrimSpace(ctx.Query(key))
	if raw == "" {
		return defaultValue
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 0 {
		return defaultValue
	}
	return v
}
