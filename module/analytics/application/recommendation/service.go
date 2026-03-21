package recommendation

import (
	"context"
	"errors"
	"sort"
	"strings"

	"mannaiah/module/analytics/domain"
	"mannaiah/module/analytics/port"
)

var (
	// ErrNilAffinityStore is returned when a nil affinity store dependency is provided.
	ErrNilAffinityStore = errors.New("affinity store must not be nil")
	// ErrNilCorrelationStore is returned when a nil tag correlation store dependency is provided.
	ErrNilCorrelationStore = errors.New("tag correlation store must not be nil")
	// ErrNilCatalogStore is returned when a nil product catalog store dependency is provided.
	ErrNilCatalogStore = errors.New("product catalog store must not be nil")
	// ErrEmptyBaseTag is returned when a recommendation query has no base tag.
	ErrEmptyBaseTag = errors.New("base tag must not be empty")
)

// Service defines recommendation use-case behavior.
type Service interface {
	// Recommend returns ranked product recommendations for one contact.
	Recommend(ctx context.Context, contactID string, query domain.RecommendationQuery) ([]domain.RecommendedProduct, error)
}

// RecommendationService implements per-contact product recommendation.
type RecommendationService struct {
	// affinityStore retrieves contact tag affinity scores from ClickHouse.
	affinityStore port.AffinityStore
	// correlationStore retrieves tag correlations from MySQL.
	correlationStore port.TagCorrelationStore
	// catalogStore retrieves product catalog entries from MySQL.
	catalogStore port.ProductCatalogStore
	// assetResolver resolves asset IDs to public URLs.
	assetResolver port.AssetURLResolver
}

var _ Service = (*RecommendationService)(nil)

// NewService creates recommendation services with required dependencies.
func NewService(
	affinityStore port.AffinityStore,
	correlationStore port.TagCorrelationStore,
	catalogStore port.ProductCatalogStore,
) (*RecommendationService, error) {
	if affinityStore == nil {
		return nil, ErrNilAffinityStore
	}
	if correlationStore == nil {
		return nil, ErrNilCorrelationStore
	}
	if catalogStore == nil {
		return nil, ErrNilCatalogStore
	}

	return &RecommendationService{
		affinityStore:    affinityStore,
		correlationStore: correlationStore,
		catalogStore:     catalogStore,
		assetResolver:    port.NoopAssetURLResolver{},
	}, nil
}

// SetAssetResolver configures asset URL resolution dependencies.
func (s *RecommendationService) SetAssetResolver(resolver port.AssetURLResolver) {
	if s == nil {
		return
	}
	if resolver == nil {
		s.assetResolver = port.NoopAssetURLResolver{}
		return
	}

	s.assetResolver = resolver
}

// Recommend returns ranked product recommendations for one contact.
//
// Resolution steps:
//  1. If UseContactAffinity is set, fetch top tag affinities for the contact.
//  2. Expand affinity tags via tag_correlations (cross-sell expansion).
//  3. Query product catalog for candidates with BaseTag and expanded tags.
//  4. Rank candidates by summed affinity score of their tags.
//  5. Resolve realm-aware display data (name, image URL) for top Limit results.
func (s *RecommendationService) Recommend(ctx context.Context, contactID string, query domain.RecommendationQuery) ([]domain.RecommendedProduct, error) {
	query.Normalize()

	if query.BaseTag == "" {
		return nil, ErrEmptyBaseTag
	}

	// Step 1: resolve contact tag affinity scores.
	var affinityScores map[string]float64
	var expandedTags []string

	if query.UseContactAffinity && contactID != "" {
		tagAffinities, err := s.affinityStore.GetTagAffinity(ctx, contactID, 20, query.AffinityMinScorePct)
		if err != nil {
			return nil, err
		}

		if len(tagAffinities) > 0 {
			affinityScores = make(map[string]float64, len(tagAffinities))
			sourceTags := make([]string, 0, len(tagAffinities))
			for _, ta := range tagAffinities {
				affinityScores[ta.Tag] = ta.AffinityScore
				sourceTags = append(sourceTags, ta.Tag)
			}

			// Step 2: expand via tag correlations.
			correlations, err := s.correlationStore.GetCorrelations(ctx, sourceTags)
			if err != nil {
				return nil, err
			}

			seen := make(map[string]struct{}, len(correlations))
			expandedTags = make([]string, 0, len(correlations))
			for _, c := range correlations {
				if _, ok := seen[c.TargetTag]; !ok {
					seen[c.TargetTag] = struct{}{}
					expandedTags = append(expandedTags, c.TargetTag)
				}
			}
		}
	}

	// Step 3: fetch product candidates.
	candidates, err := s.catalogStore.GetProductsByBaseTag(ctx, query.BaseTag, expandedTags, query.CategoryID, query.Limit*3)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	// Step 4: rank by summed affinity score of product tags.
	if affinityScores != nil {
		sort.Slice(candidates, func(i, j int) bool {
			return productAffinityScore(candidates[i].Tags, affinityScores) >
				productAffinityScore(candidates[j].Tags, affinityScores)
		})
	}

	if len(candidates) > query.Limit {
		candidates = candidates[:query.Limit]
	}

	// Step 5: resolve display data.
	results := make([]domain.RecommendedProduct, 0, len(candidates))
	for _, c := range candidates {
		name := resolveDatasheetName(c.Datasheets, query.Realm)
		imageURL := resolveGalleryImage(ctx, c.Gallery, query.Realm, s.assetResolver)

		results = append(results, domain.RecommendedProduct{
			ID:       c.ID,
			Name:     name,
			Price:    c.Price,
			ImageURL: imageURL,
		})
	}

	return results, nil
}

// productAffinityScore sums affinity scores for all tags a product carries.
func productAffinityScore(tags []string, scores map[string]float64) float64 {
	var total float64
	for _, t := range tags {
		total += scores[t]
	}

	return total
}

// resolveDatasheetName returns the first datasheet name matching the realm,
// falling back to the first datasheet name if no realm match is found.
func resolveDatasheetName(datasheets []port.ProductDatasheetEntry, realm string) string {
	var fallback string
	for _, d := range datasheets {
		if fallback == "" {
			fallback = d.Name
		}
		if strings.EqualFold(d.Realm, realm) {
			return d.Name
		}
	}

	return fallback
}

// resolveGalleryImage returns the resolved URL for the first gallery image
// that is visible in the requested realm. Falls back to the main image,
// then the first image, if no realm match is found.
func resolveGalleryImage(ctx context.Context, gallery []port.ProductGalleryEntry, realm string, resolver port.AssetURLResolver) string {
	var mainAsset, firstAsset string

	for _, g := range gallery {
		if firstAsset == "" {
			firstAsset = g.AssetID
		}
		if g.IsMain && mainAsset == "" {
			mainAsset = g.AssetID
		}

		// An empty IncludedRealms slice means the image is visible in all realms.
		if len(g.IncludedRealms) == 0 {
			if g.IsMain {
				return resolver.ResolveURL(ctx, g.AssetID)
			}
			continue
		}

		for _, r := range g.IncludedRealms {
			if strings.EqualFold(r, realm) {
				return resolver.ResolveURL(ctx, g.AssetID)
			}
		}
	}

	if mainAsset != "" {
		return resolver.ResolveURL(ctx, mainAsset)
	}
	if firstAsset != "" {
		return resolver.ResolveURL(ctx, firstAsset)
	}

	return ""
}
