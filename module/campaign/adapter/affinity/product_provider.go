package affinity

import (
	"context"
	"errors"

	"mannaiah/module/analytics/application/recommendation"
	"mannaiah/module/analytics/domain"
	campaigndomain "mannaiah/module/campaign/domain"
)

var (
	// ErrNilRecommendationService is returned when a nil recommendation service dependency is provided.
	ErrNilRecommendationService = errors.New("recommendation service must not be nil")
)

// ProductProvider adapts the analytics RecommendationService for use as an AffinityProductProvider
// within the campaign module. It maps campaign ProductBlock values to RecommendationQuery values,
// calls the analytics service, and maps the results to campaign TemplateProduct values.
type ProductProvider struct {
	// service defines the analytics recommendation service dependency.
	service *recommendation.RecommendationService
}

// NewProductProvider creates campaign affinity product providers.
func NewProductProvider(service *recommendation.RecommendationService) (*ProductProvider, error) {
	if service == nil {
		return nil, ErrNilRecommendationService
	}

	return &ProductProvider{service: service}, nil
}

// GetProducts returns recommended products for one contact and one product block configuration.
func (p *ProductProvider) GetProducts(ctx context.Context, contactID string, block campaigndomain.ProductBlock) ([]campaigndomain.TemplateProduct, error) {
	query := domain.RecommendationQuery{
		BaseTag:             block.BaseTag,
		BaseTags:            block.BaseTags,
		BaseTagMode:         block.BaseTagMode,
		UseContactAffinity:  block.UseAffinity,
		AffinityMinScorePct: block.AffinityMinScorePct,
		CategoryID:          block.CategoryID,
		Realm:               block.Realm,
		Limit:               block.Limit,
		PinnedProductIDs:    block.PinnedProductIDs,
		ExcludeProductIDs:   block.ExcludeProductIDs,
		FilterVariationIDs:  block.FilterVariationIDs,
		PreferVariationIDs:  block.PreferVariationIDs,
	}

	recommended, err := p.service.Recommend(ctx, contactID, query)
	if err != nil {
		return nil, err
	}

	result := make([]campaigndomain.TemplateProduct, 0, len(recommended))
	for _, r := range recommended {
		result = append(result, campaigndomain.TemplateProduct{
			ID:       r.ID,
			Name:     r.Name,
			Price:    r.Price,
			ImageURL: r.ImageURL,
			URL:      r.URL,
		})
	}

	return result, nil
}
