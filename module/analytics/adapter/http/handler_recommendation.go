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
//   - baseTag     (required) product base tag to filter by
//   - categoryId  (optional) restrict to one category
//   - realm       (optional) display realm, default "default"
//   - limit       (optional) max results [1,10], default 3
//   - affinity    (optional) "true" to enable contact-affinity filtering
//   - minScore    (optional) minimum affinity score percentile [0,100], default 0
func (h *RecommendationHandler) getRecommendations(ctx corehttp.Context) error {
	contactID := strings.TrimSpace(ctx.Params("contactId"))

	baseTag := strings.TrimSpace(ctx.Query("baseTag"))
	if baseTag == "" {
		return corehttp.NewAppError(400, "baseTag query parameter is required", nil)
	}

	useAffinity := strings.EqualFold(strings.TrimSpace(ctx.Query("affinity")), "true")

	query := domain.RecommendationQuery{
		BaseTag:             baseTag,
		UseContactAffinity:  useAffinity,
		AffinityMinScorePct: queryFloat64(ctx, "minScore", 0),
		CategoryID:          strings.TrimSpace(ctx.Query("categoryId")),
		Realm:               strings.TrimSpace(ctx.Query("realm")),
		Limit:               queryInt(ctx, "limit", 3),
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

// mapRecommendationError maps recommendation service errors to HTTP-layer app errors.
func (h *RecommendationHandler) mapRecommendationError(err error) error {
	if h != nil && h.authorizer != nil {
		if err2 := mapAuthError(h.authorizer, err); err2 != err {
			return err2
		}
	}

	return mapServiceError(err)
}
