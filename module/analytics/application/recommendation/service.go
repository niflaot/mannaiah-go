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
	// ErrEmptyBaseTag is returned when a recommendation query has no base tag and no pinned products.
	ErrEmptyBaseTag = errors.New("base tag must not be empty unless pinned product IDs are provided")
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
//  1. Load pinned products by ID (always first, bypass affinity/base-tag filter).
//  2. Build the exclusion set: ExcludeProductIDs ∪ PinnedProductIDs.
//  3. If BaseTag is set, fetch contact tag affinity and expand via tag_correlations.
//  4. Query dynamic candidates (excluded IDs removed), rank by affinity score.
//  5. Combine: pinned first, then dynamic up to Limit total.
//  6. For each product: resolve realm price and optional realm image — skip only when price is missing.
func (s *RecommendationService) Recommend(ctx context.Context, contactID string, query domain.RecommendationQuery) ([]domain.RecommendedProduct, error) {
	query.Normalize()

	if len(query.BaseTags) == 0 && len(query.PinnedProductIDs) == 0 {
		return nil, ErrEmptyBaseTag
	}

	// Step 1: load pinned products.
	var pinnedEntries []port.ProductCatalogEntry
	if len(query.PinnedProductIDs) > 0 {
		var err error
		pinnedEntries, err = s.catalogStore.GetProductsByIDs(ctx, query.PinnedProductIDs, query.FilterVariationIDs)
		if err != nil {
			return nil, err
		}
	}

	// Step 2: build unified exclusion set (pinned IDs + explicit exclude IDs).
	excludeSet := make(map[string]struct{}, len(query.ExcludeProductIDs)+len(pinnedEntries))
	for _, id := range query.ExcludeProductIDs {
		excludeSet[id] = struct{}{}
	}
	for _, e := range pinnedEntries {
		excludeSet[e.ID] = struct{}{}
	}
	excludeIDs := make([]string, 0, len(excludeSet))
	for id := range excludeSet {
		excludeIDs = append(excludeIDs, id)
	}

	// How many dynamic slots remain after pinned products.
	dynamicLimit := query.Limit - len(pinnedEntries)

	var dynamicEntries []port.ProductCatalogEntry

	if dynamicLimit > 0 && len(query.BaseTags) > 0 {
		// Step 3: resolve contact tag affinity scores and expand via correlations.
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

		// Step 4: fetch and rank dynamic candidates.
		candidates, err := s.catalogStore.GetProductsByBaseTags(ctx, query.BaseTags, query.BaseTagMode, expandedTags, query.CategoryID, excludeIDs, query.FilterVariationIDs, dynamicLimit*3)
		if err != nil {
			return nil, err
		}

		if affinityScores != nil {
			sort.Slice(candidates, func(i, j int) bool {
				return productAffinityScore(candidates[i].Tags, affinityScores) >
					productAffinityScore(candidates[j].Tags, affinityScores)
			})
		}

		if len(candidates) > dynamicLimit {
			candidates = candidates[:dynamicLimit]
		}
		dynamicEntries = candidates
	}

	// Step 5: combine pinned + dynamic.
	all := make([]port.ProductCatalogEntry, 0, len(pinnedEntries)+len(dynamicEntries))
	all = append(all, pinnedEntries...)
	all = append(all, dynamicEntries...)

	if len(all) == 0 {
		return nil, nil
	}

	// Step 6: resolve realm price and optional image; skip products missing price.
	results := make([]domain.RecommendedProduct, 0, len(all))
	for _, c := range all {
		price, hasPrice := resolveRealmPrice(c.Datasheets, query.Realm)
		imageURL, _ := resolveRealmImage(ctx, c.Gallery, query.Realm, query.PreferVariationIDs, s.assetResolver)
		if !hasPrice {
			continue
		}
		urlVariationCandidates := resolveURLVariationCandidates(c.VariationIDs, query.PreferVariationIDs, query.FilterVariationIDs)
		results = append(results, domain.RecommendedProduct{
			ID:       c.ID,
			Name:     resolveDatasheetName(c.Datasheets, query.Realm),
			Price:    price,
			ImageURL: imageURL,
			URL:      resolveRealmURL(c.Datasheets, query.Realm, urlVariationCandidates),
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

// resolveRealmPrice returns the price from the first datasheet matching the realm.
// Returns (0, false) when no realm datasheet has a price attribute.
func resolveRealmPrice(datasheets []port.ProductDatasheetEntry, realm string) (float64, bool) {
	for _, d := range datasheets {
		if strings.EqualFold(d.Realm, realm) && d.Price != nil {
			return *d.Price, true
		}
	}

	return 0, false
}

// resolveRealmImage returns the URL of the best gallery image visible in the requested realm.
//
// Selection order:
//  1. First realm-visible image linked to a preferred variation (if PreferVariationIDs is set).
//  2. First realm-visible image regardless of variation.
//
// An image is "visible in realm R" when its IncludedRealms is empty (all realms) or contains R.
// Returns ("", false) when no realm-visible image URL can be resolved.
func resolveRealmImage(ctx context.Context, gallery []port.ProductGalleryEntry, realm string, preferVariationIDs []string, resolver port.AssetURLResolver) (string, bool) {
	// Pass 1: prefer variation-specific image visible in the realm.
	if len(preferVariationIDs) > 0 {
		preferSet := make(map[string]struct{}, len(preferVariationIDs))
		for _, v := range preferVariationIDs {
			preferSet[v] = struct{}{}
		}
		for _, g := range gallery {
			if !isVisibleInRealm(g, realm) {
				continue
			}
			for _, vid := range g.VariationIDs {
				if _, ok := preferSet[vid]; ok {
					if url := resolveGalleryImageURL(ctx, g, resolver); url != "" {
						return url, true
					}
				}
			}
		}
	}

	// Pass 2: first realm-visible image regardless of variation.
	for _, g := range gallery {
		if isVisibleInRealm(g, realm) {
			if url := resolveGalleryImageURL(ctx, g, resolver); url != "" {
				return url, true
			}
		}
	}

	return "", false
}

// resolveGalleryImageURL resolves one gallery image URL from resolver and metadata fallbacks.
func resolveGalleryImageURL(ctx context.Context, galleryItem port.ProductGalleryEntry, resolver port.AssetURLResolver) string {
	if resolved := strings.TrimSpace(resolver.ResolveURL(ctx, galleryItem.AssetID)); resolved != "" {
		return resolved
	}
	if resolved := strings.TrimSpace(galleryItem.AssetURL); resolved != "" {
		return resolved
	}
	assetID := strings.TrimSpace(galleryItem.AssetID)
	if strings.HasPrefix(strings.ToLower(assetID), "http://") || strings.HasPrefix(strings.ToLower(assetID), "https://") {
		return assetID
	}

	return ""
}

// isVisibleInRealm reports whether a gallery entry is visible in the given realm.
// A gallery item with no IncludedRealms rows is visible everywhere.
func isVisibleInRealm(g port.ProductGalleryEntry, realm string) bool {
	if len(g.IncludedRealms) == 0 {
		return true
	}
	for _, r := range g.IncludedRealms {
		if strings.EqualFold(r, realm) {
			return true
		}
	}

	return false
}

// resolveRealmURL returns the first product URL matching the requested realm.
// When variationCandidates are provided, "<variation>.url" values are preferred over plain "url".
// Returns empty when no URL is available for the requested realm.
func resolveRealmURL(datasheets []port.ProductDatasheetEntry, realm string, variationCandidates []string) string {
	for _, datasheet := range datasheets {
		if !strings.EqualFold(datasheet.Realm, realm) {
			continue
		}
		url := resolveDatasheetURL(datasheet, variationCandidates)
		if url != "" {
			return url
		}
	}

	return ""
}

// resolveDatasheetURL resolves one datasheet URL, preferring variation-scoped entries.
func resolveDatasheetURL(datasheet port.ProductDatasheetEntry, variationCandidates []string) string {
	if len(variationCandidates) > 0 && len(datasheet.VariationURLs) > 0 {
		for _, variationID := range variationCandidates {
			url := strings.TrimSpace(datasheet.VariationURLs[variationID])
			if url != "" {
				return url
			}
		}
	}

	return strings.TrimSpace(datasheet.URL)
}

// resolveURLVariationCandidates builds ordered variation candidates for scoped URL lookup.
// Order: preferVariationIDs -> filterVariationIDs -> product variation links.
func resolveURLVariationCandidates(productVariationIDs []string, preferVariationIDs []string, filterVariationIDs []string) []string {
	if len(productVariationIDs) == 0 {
		return nil
	}

	productSet := make(map[string]struct{}, len(productVariationIDs))
	orderedProductVariations := make([]string, 0, len(productVariationIDs))
	for _, rawVariationID := range productVariationIDs {
		variationID := normalizeVariationToken(rawVariationID)
		if variationID == "" {
			continue
		}
		if _, exists := productSet[variationID]; exists {
			continue
		}
		productSet[variationID] = struct{}{}
		orderedProductVariations = append(orderedProductVariations, variationID)
	}
	if len(orderedProductVariations) == 0 {
		return nil
	}

	result := make([]string, 0, len(orderedProductVariations))
	seen := make(map[string]struct{}, len(orderedProductVariations))
	appendCandidates := func(source []string) {
		for _, rawVariationID := range source {
			variationID := normalizeVariationToken(rawVariationID)
			if variationID == "" {
				continue
			}
			if _, allowed := productSet[variationID]; !allowed {
				continue
			}
			if _, alreadyIncluded := seen[variationID]; alreadyIncluded {
				continue
			}
			seen[variationID] = struct{}{}
			result = append(result, variationID)
		}
	}
	appendCandidates(preferVariationIDs)
	appendCandidates(filterVariationIDs)
	appendCandidates(orderedProductVariations)

	return result
}

// normalizeVariationToken resolves trimmed lower-case variation tokens.
func normalizeVariationToken(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
