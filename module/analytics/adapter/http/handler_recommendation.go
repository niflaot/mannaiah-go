package http

import (
	"context"
	"strings"

	"mannaiah/module/analytics/domain"
	corehttp "mannaiah/module/core/http"
)

// RecommendationService defines recommendation use-case behavior required by HTTP handlers.
type RecommendationService interface {
	// Recommend returns ranked product recommendations for one contact.
	Recommend(ctx context.Context, contactID string, query domain.RecommendationQuery) ([]domain.RecommendedProduct, error)
}

// RecommendationHandler defines HTTP route handlers for recommendation endpoints.
type RecommendationHandler struct {
	// service defines recommendation use-case dependencies.
	service RecommendationService
	// authorizer defines optional auth dependencies.
	authorizer Authorizer
}

// NewRecommendationHandler creates recommendation HTTP handlers.
func NewRecommendationHandler(service RecommendationService, authorizers ...Authorizer) *RecommendationHandler {
	var authorizer Authorizer
	if len(authorizers) > 0 {
		authorizer = authorizers[0]
	}

	return &RecommendationHandler{service: service, authorizer: authorizer}
}

// SetAuthorizer configures endpoint authentication and permission dependencies.
func (h *RecommendationHandler) SetAuthorizer(authorizer Authorizer) {
	if h == nil {
		return
	}
	h.authorizer = authorizer
}

// RegisterRoutes registers recommendation routes on the provided router.
func (h *RecommendationHandler) RegisterRoutes(router corehttp.Router) {
	router.Get("/analytics/recommendations/contacts/:contactId", h.protect("marketing:manage", h.getRecommendations))
}

// getRecommendations handles GET /analytics/recommendations/contacts/:contactId.
//
// Query parameters:
//   - baseTag              (backward compat) single base tag; merged into baseTags
//   - baseTags             comma-separated base tags (required unless pinnedIds is set)
//   - baseTagMode          "any" (union, default) or "all" (intersection) for baseTags matching
//   - categoryId           (optional) restrict to one category
//   - categoryIds          (optional) comma-separated include category references
//   - excludeCategoryIds   (optional) comma-separated exclude category references
//   - includeTags          (optional) comma-separated include-tag filter values (OR semantics)
//   - excludeTags          (optional) comma-separated exclude-tag filter values
//   - minPrice             (optional) minimum product price filter
//   - maxPrice             (optional) maximum product price filter
//   - excludePurchased     (optional) "true" to exclude products already purchased by the contact
//   - realm                (optional) display realm, default "default"
//   - limit                (optional) max results [1,10], default 3
//   - affinity             (optional) "true" to enable contact-affinity filtering
//   - minScore             (optional) minimum affinity score percentile [0,100], default 0
//   - pinnedIds            (optional) comma-separated product IDs always shown first
//   - excludeIds           (optional) comma-separated product IDs never shown
//   - filterVariationIds   (optional) comma-separated variation IDs; only products with ≥1 match returned
//   - preferVariationIds   (optional) comma-separated variation IDs; prefer gallery images for these variations
func (h *RecommendationHandler) getRecommendations(ctx corehttp.Context) error {
	contactID := strings.TrimSpace(ctx.Params("contactId"))

	baseTag := strings.TrimSpace(ctx.Query("baseTag"))
	baseTags := splitCommaSeparated(ctx.Query("baseTags"))
	pinnedIDs := splitCommaSeparated(ctx.Query("pinnedIds"))
	excludeIDs := splitCommaSeparated(ctx.Query("excludeIds"))

	if baseTag == "" && len(baseTags) == 0 && len(pinnedIDs) == 0 {
		return corehttp.NewAppError(400, "baseTag or baseTags query parameter is required when pinnedIds is not set", nil)
	}

	useAffinity := strings.EqualFold(strings.TrimSpace(ctx.Query("affinity")), "true")

	query := domain.RecommendationQuery{
		BaseTag:                  baseTag,
		BaseTags:                 baseTags,
		BaseTagMode:              strings.TrimSpace(ctx.Query("baseTagMode")),
		UseContactAffinity:       useAffinity,
		AffinityMinScorePct:      queryFloat64(ctx, "minScore", 0),
		CategoryID:               strings.TrimSpace(ctx.Query("categoryId")),
		CategoryIDs:              splitCommaSeparated(ctx.Query("categoryIds")),
		ExcludeCategoryIDs:       splitCommaSeparated(ctx.Query("excludeCategoryIds")),
		IncludeTags:              splitCommaSeparated(ctx.Query("includeTags")),
		ExcludeTags:              splitCommaSeparated(ctx.Query("excludeTags")),
		MinPrice:                 queryOptionalFloat64(ctx, "minPrice"),
		MaxPrice:                 queryOptionalFloat64(ctx, "maxPrice"),
		ExcludePurchasedProducts: strings.EqualFold(strings.TrimSpace(ctx.Query("excludePurchased")), "true"),
		Realm:                    strings.TrimSpace(ctx.Query("realm")),
		Limit:                    queryInt(ctx, "limit", 3),
		PinnedProductIDs:         pinnedIDs,
		ExcludeProductIDs:        excludeIDs,
		FilterVariationIDs:       splitCommaSeparated(ctx.Query("filterVariationIds")),
		PreferVariationIDs:       splitCommaSeparated(ctx.Query("preferVariationIds")),
	}

	products, err := h.service.Recommend(ctx.Context(), contactID, query)
	if err != nil {
		return h.mapRecommendationError(err)
	}

	if products == nil {
		products = []domain.RecommendedProduct{}
	}

	return ctx.Status(200).JSON(products)
}

// protect wraps recommendation endpoint handlers with optional authentication.
func (h *RecommendationHandler) protect(permission string, next corehttp.Handler) corehttp.Handler {
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

// splitCommaSeparated splits a comma-separated query parameter into trimmed non-empty strings.
func splitCommaSeparated(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			result = append(result, s)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// queryOptionalFloat64 parses optional float64 query parameter values.
func queryOptionalFloat64(ctx corehttp.Context, key string) *float64 {
	value := strings.TrimSpace(ctx.Query(key))
	if value == "" {
		return nil
	}
	parsed := queryFloat64(ctx, key, -1)
	if parsed < 0 {
		return nil
	}

	return &parsed
}

// mapRecommendationError maps recommendation service errors to HTTP-layer app errors.
func (h *RecommendationHandler) mapRecommendationError(err error) error {
	if h != nil && h.authorizer != nil {
		if err2 := mapAuthError(h.authorizer, err); err2 != err {
			return err2
		}
	}

	return mapServiceError(err)
}
