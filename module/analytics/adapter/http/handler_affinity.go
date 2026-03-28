package http

import (
	"context"
	"strings"

	"mannaiah/module/analytics/domain"
	corehttp "mannaiah/module/core/http"
)

// AffinityService defines affinity use-case behavior required by HTTP handlers.
type AffinityService interface {
	// GetTagAffinity retrieves ranked tag affinity scores for one contact.
	GetTagAffinity(ctx context.Context, contactID string, limit int, minScore float64) ([]domain.TagAffinity, error)
	// GetCategoryAffinity retrieves ranked category affinity scores for one contact.
	GetCategoryAffinity(ctx context.Context, contactID string, limit int, minScore float64) ([]domain.CategoryAffinity, error)
	// GetVariationAffinity retrieves ranked variation affinity scores for one contact.
	GetVariationAffinity(ctx context.Context, contactID string, limit int, minScore float64) ([]domain.VariationAffinity, error)
	// GetProfile assembles a full affinity profile for one contact.
	GetProfile(ctx context.Context, contactID string, limit int, minScore float64) (*domain.AffinityProfile, error)
	// RefreshAll truncates and repopulates all affinity materialized views.
	RefreshAll(ctx context.Context) error
}

// AffinityHandler defines HTTP route handlers for affinity endpoints.
type AffinityHandler struct {
	// service defines affinity use-case dependencies.
	service AffinityService
	// authorizer defines optional auth dependencies.
	authorizer Authorizer
}

// NewAffinityHandler creates affinity HTTP handlers.
func NewAffinityHandler(service AffinityService, authorizers ...Authorizer) *AffinityHandler {
	var authorizer Authorizer
	if len(authorizers) > 0 {
		authorizer = authorizers[0]
	}

	return &AffinityHandler{service: service, authorizer: authorizer}
}

// SetAuthorizer configures endpoint authentication and permission dependencies.
func (h *AffinityHandler) SetAuthorizer(authorizer Authorizer) {
	if h == nil {
		return
	}
	h.authorizer = authorizer
}

// RegisterRoutes registers affinity routes on the provided router.
func (h *AffinityHandler) RegisterRoutes(router corehttp.Router) {
	router.Get("/analytics/affinity/contacts/:contactId", h.protect("marketing:manage", h.getProfile))
	router.Get("/analytics/affinity/contacts/:contactId/tags", h.protect("marketing:manage", h.getTagAffinity))
	router.Get("/analytics/affinity/contacts/:contactId/categories", h.protect("marketing:manage", h.getCategoryAffinity))
	router.Get("/analytics/affinity/contacts/:contactId/variations", h.protect("marketing:manage", h.getVariationAffinity))
	router.Post("/analytics/affinity/refresh", h.protect("marketing:manage", h.refreshAll))
}

// getProfile handles GET /analytics/affinity/contacts/:contactId.
func (h *AffinityHandler) getProfile(ctx corehttp.Context) error {
	contactID := strings.TrimSpace(ctx.Params("contactId"))
	limit := queryInt(ctx, "limit", 10)
	minScore := queryFloat64(ctx, "minScore", 0)

	profile, err := h.service.GetProfile(ctx.Context(), contactID, limit, minScore)
	if err != nil {
		return h.mapAffinityError(err)
	}

	return ctx.Status(200).JSON(profile)
}

// getTagAffinity handles GET /analytics/affinity/contacts/:contactId/tags.
func (h *AffinityHandler) getTagAffinity(ctx corehttp.Context) error {
	contactID := strings.TrimSpace(ctx.Params("contactId"))
	limit := queryInt(ctx, "limit", 10)
	minScore := queryFloat64(ctx, "minScore", 0)

	rows, err := h.service.GetTagAffinity(ctx.Context(), contactID, limit, minScore)
	if err != nil {
		return h.mapAffinityError(err)
	}

	return ctx.Status(200).JSON(rows)
}

// getCategoryAffinity handles GET /analytics/affinity/contacts/:contactId/categories.
func (h *AffinityHandler) getCategoryAffinity(ctx corehttp.Context) error {
	contactID := strings.TrimSpace(ctx.Params("contactId"))
	limit := queryInt(ctx, "limit", 10)
	minScore := queryFloat64(ctx, "minScore", 0)

	rows, err := h.service.GetCategoryAffinity(ctx.Context(), contactID, limit, minScore)
	if err != nil {
		return h.mapAffinityError(err)
	}

	return ctx.Status(200).JSON(rows)
}

// getVariationAffinity handles GET /analytics/affinity/contacts/:contactId/variations.
func (h *AffinityHandler) getVariationAffinity(ctx corehttp.Context) error {
	contactID := strings.TrimSpace(ctx.Params("contactId"))
	limit := queryInt(ctx, "limit", 10)
	minScore := queryFloat64(ctx, "minScore", 0)

	rows, err := h.service.GetVariationAffinity(ctx.Context(), contactID, limit, minScore)
	if err != nil {
		return h.mapAffinityError(err)
	}

	return ctx.Status(200).JSON(rows)
}

// refreshAll handles POST /analytics/affinity/refresh.
func (h *AffinityHandler) refreshAll(ctx corehttp.Context) error {
	if err := h.service.RefreshAll(ctx.Context()); err != nil {
		return h.mapAffinityError(err)
	}

	return ctx.Status(200).JSON(map[string]string{"status": "ok"})
}

// protect wraps affinity endpoint handlers with optional authentication.
func (h *AffinityHandler) protect(permission string, next corehttp.Handler) corehttp.Handler {
	if h == nil || h.authorizer == nil {
		return next
	}

	return func(ctx corehttp.Context) error {
		if err := h.authorizer.Require(ctx.Context(), ctx.GetHeader("Authorization"), permission); err != nil {
			return mapAuthError(h.authorizer, err)
		}

		return next(ctx)
	}
}

// mapAffinityError maps affinity service errors to HTTP-layer app errors.
func (h *AffinityHandler) mapAffinityError(err error) error {
	if h != nil && h.authorizer != nil {
		if err2 := mapAuthError(h.authorizer, err); err2 != err {
			return err2
		}
	}

	return mapServiceError(err)
}

// queryInt parses an integer query parameter with a default fallback.
func queryInt(ctx corehttp.Context, key string, defaultVal int) int {
	val := strings.TrimSpace(ctx.Query(key))
	if val == "" {
		return defaultVal
	}
	n := 0
	for _, ch := range val {
		if ch < '0' || ch > '9' {
			return defaultVal
		}
		n = n*10 + int(ch-'0')
	}

	return n
}

// queryInt64 parses an int64 query parameter with a default fallback.
// Supports optional leading minus sign for negative values.
func queryInt64(ctx corehttp.Context, key string, defaultVal int64) int64 {
	val := strings.TrimSpace(ctx.Query(key))
	if val == "" {
		return defaultVal
	}
	negative := false
	start := 0
	if len(val) > 0 && val[0] == '-' {
		negative = true
		start = 1
	}
	if start >= len(val) {
		return defaultVal
	}
	var n int64
	for _, ch := range val[start:] {
		if ch < '0' || ch > '9' {
			return defaultVal
		}
		n = n*10 + int64(ch-'0')
	}
	if negative {
		n = -n
	}
	return n
}

// queryFloat64 parses a float64 query parameter with a default fallback.
func queryFloat64(ctx corehttp.Context, key string, defaultVal float64) float64 {
	val := strings.TrimSpace(ctx.Query(key))
	if val == "" {
		return defaultVal
	}
	var f float64
	var frac float64
	var inFrac bool
	var fracDiv float64 = 1
	for _, ch := range val {
		if ch == '.' {
			inFrac = true
			continue
		}
		if ch < '0' || ch > '9' {
			return defaultVal
		}
		digit := float64(ch - '0')
		if inFrac {
			fracDiv *= 10
			frac += digit / fracDiv
		} else {
			f = f*10 + digit
		}
	}

	return f + frac
}
