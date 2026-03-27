package recommendation

import (
	"context"
	"reflect"
	"testing"

	"mannaiah/module/analytics/domain"
	"mannaiah/module/analytics/port"
)

type affinityStoreStub struct{}

// GetTagAffinity returns empty affinity rows.
func (affinityStoreStub) GetTagAffinity(_ context.Context, _ string, _ int, _ float64) ([]domain.TagAffinity, error) {
	return nil, nil
}

// GetCategoryAffinity returns empty affinity rows.
func (affinityStoreStub) GetCategoryAffinity(_ context.Context, _ string, _ int, _ float64) ([]domain.CategoryAffinity, error) {
	return nil, nil
}

// GetVariationAffinity returns empty affinity rows.
func (affinityStoreStub) GetVariationAffinity(_ context.Context, _ string, _ int, _ float64) ([]domain.VariationAffinity, error) {
	return nil, nil
}

// GetProfile returns an empty affinity profile.
func (affinityStoreStub) GetProfile(_ context.Context, _ string, _ int, _ float64) (*domain.AffinityProfile, error) {
	return &domain.AffinityProfile{}, nil
}

// GetPurchasedProductIDs returns no purchased product IDs.
func (affinityStoreStub) GetPurchasedProductIDs(_ context.Context, _ string, _ int) ([]string, error) {
	return nil, nil
}

type affinityStorePurchasedStub struct {
	purchasedProductIDs []string
}

// GetTagAffinity returns empty affinity rows.
func (s affinityStorePurchasedStub) GetTagAffinity(_ context.Context, _ string, _ int, _ float64) ([]domain.TagAffinity, error) {
	return nil, nil
}

// GetCategoryAffinity returns empty affinity rows.
func (s affinityStorePurchasedStub) GetCategoryAffinity(_ context.Context, _ string, _ int, _ float64) ([]domain.CategoryAffinity, error) {
	return nil, nil
}

// GetVariationAffinity returns empty affinity rows.
func (s affinityStorePurchasedStub) GetVariationAffinity(_ context.Context, _ string, _ int, _ float64) ([]domain.VariationAffinity, error) {
	return nil, nil
}

// GetProfile returns an empty affinity profile.
func (s affinityStorePurchasedStub) GetProfile(_ context.Context, _ string, _ int, _ float64) (*domain.AffinityProfile, error) {
	return &domain.AffinityProfile{}, nil
}

// GetPurchasedProductIDs returns predefined purchased product IDs.
func (s affinityStorePurchasedStub) GetPurchasedProductIDs(_ context.Context, _ string, _ int) ([]string, error) {
	return append([]string(nil), s.purchasedProductIDs...), nil
}

// RefreshTagMV is a no-op for tests.
func (s affinityStorePurchasedStub) RefreshTagMV(_ context.Context) error { return nil }

// RefreshCategoryMV is a no-op for tests.
func (s affinityStorePurchasedStub) RefreshCategoryMV(_ context.Context) error { return nil }

// RefreshVariationMV is a no-op for tests.
func (s affinityStorePurchasedStub) RefreshVariationMV(_ context.Context) error { return nil }

// RefreshTagMV is a no-op for tests.
func (affinityStoreStub) RefreshTagMV(_ context.Context) error { return nil }

// RefreshCategoryMV is a no-op for tests.
func (affinityStoreStub) RefreshCategoryMV(_ context.Context) error { return nil }

// RefreshVariationMV is a no-op for tests.
func (affinityStoreStub) RefreshVariationMV(_ context.Context) error { return nil }

type correlationStoreStub struct{}

// GetCorrelations returns no correlations.
func (correlationStoreStub) GetCorrelations(_ context.Context, _ []string) ([]port.TagCorrelation, error) {
	return nil, nil
}

type catalogStoreStub struct {
	entries              []port.ProductCatalogEntry
	entriesByID          map[string]port.ProductCatalogEntry
	lastIDsQuery         []string
	lastIDsVariationIDs  []string
	lastCategoryIDFilter string
	lastMinPriceFilter   *float64
	lastMaxPriceFilter   *float64
}

// GetProductsByBaseTags returns predefined test entries.
func (s *catalogStoreStub) GetProductsByBaseTags(
	_ context.Context,
	_ []string,
	_ string,
	_ []string,
	categoryID string,
	_ []string,
	_ []string,
	_ []string,
	_ []string,
	minPrice *float64,
	maxPrice *float64,
	excludeIDs []string,
	_ []string,
	_ int,
) ([]port.ProductCatalogEntry, error) {
	s.lastCategoryIDFilter = categoryID
	s.lastMinPriceFilter = minPrice
	s.lastMaxPriceFilter = maxPrice
	if len(excludeIDs) == 0 {
		return s.entries, nil
	}
	excludeSet := make(map[string]struct{}, len(excludeIDs))
	for _, id := range excludeIDs {
		excludeSet[id] = struct{}{}
	}
	filtered := make([]port.ProductCatalogEntry, 0, len(s.entries))
	for _, entry := range s.entries {
		if _, excluded := excludeSet[entry.ID]; excluded {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered, nil
}

// GetProductsByIDs returns configured entries preserving requested ID order.
func (s *catalogStoreStub) GetProductsByIDs(_ context.Context, ids []string, variationIDs []string) ([]port.ProductCatalogEntry, error) {
	s.lastIDsQuery = append([]string(nil), ids...)
	s.lastIDsVariationIDs = append([]string(nil), variationIDs...)
	if len(s.entriesByID) == 0 {
		return nil, nil
	}
	entries := make([]port.ProductCatalogEntry, 0, len(ids))
	for _, id := range ids {
		entry, ok := s.entriesByID[id]
		if !ok {
			continue
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// TestRecommendIncludesProductsWithoutImage verifies products are not dropped when image URLs are unavailable.
func TestRecommendIncludesProductsWithoutImage(t *testing.T) {
	t.Parallel()

	price := 129.5
	catalog := &catalogStoreStub{entries: []port.ProductCatalogEntry{
		{
			ID: "p-1",
			Datasheets: []port.ProductDatasheetEntry{{
				Realm: "default",
				Name:  "Leather Wallet",
				Price: &price,
			}},
			Gallery: nil,
		},
	}}
	svc, err := NewService(
		affinityStoreStub{},
		correlationStoreStub{},
		catalog,
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	results, err := svc.Recommend(context.Background(), "contact-1", domain.RecommendationQuery{BaseTag: "cuero", Limit: 3})
	if err != nil {
		t.Fatalf("Recommend() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].Name != "Leather Wallet" {
		t.Fatalf("results[0].Name = %q, want %q", results[0].Name, "Leather Wallet")
	}
	if results[0].ImageURL != "" {
		t.Fatalf("results[0].ImageURL = %q, want empty", results[0].ImageURL)
	}
}

// TestRecommendPinnedScopedVariationOverridesPreferredVariation verifies product-scoped variation pins override global preferred variations.
func TestRecommendPinnedScopedVariationOverridesPreferredVariation(t *testing.T) {
	t.Parallel()

	price := 79.9
	pinnedEntry := port.ProductCatalogEntry{
		ID:           "p-1",
		VariationIDs: []string{"var-blue", "var-red"},
		Datasheets: []port.ProductDatasheetEntry{{
			Realm: "default",
			Name:  "Backpack",
			Price: &price,
			VariationURLs: map[string]string{
				"var-blue": "https://store.example.com/p-1?variation=blue",
				"var-red":  "https://store.example.com/p-1?variation=red",
			},
		}},
		Gallery: []port.ProductGalleryEntry{
			{AssetID: "asset-red", AssetURL: "https://cdn.example.com/red.jpg", VariationIDs: []string{"var-red"}},
			{AssetID: "asset-blue", AssetURL: "https://cdn.example.com/blue.jpg", VariationIDs: []string{"var-blue"}},
		},
	}
	catalog := &catalogStoreStub{
		entriesByID: map[string]port.ProductCatalogEntry{
			"p-1": pinnedEntry,
		},
	}
	svc, err := NewService(affinityStoreStub{}, correlationStoreStub{}, catalog)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	results, err := svc.Recommend(context.Background(), "contact-1", domain.RecommendationQuery{
		BaseTags:           []string{"tier-1"},
		Limit:              1,
		PinnedProductIDs:   []string{"p-1|var-blue"},
		PreferVariationIDs: []string{"var-red"},
	})
	if err != nil {
		t.Fatalf("Recommend() error = %v", err)
	}
	if !reflect.DeepEqual(catalog.lastIDsQuery, []string{"p-1"}) {
		t.Fatalf("GetProductsByIDs ids = %#v, want %#v", catalog.lastIDsQuery, []string{"p-1"})
	}
	if len(catalog.lastIDsVariationIDs) != 0 {
		t.Fatalf("GetProductsByIDs variation IDs = %#v, want empty", catalog.lastIDsVariationIDs)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].URL != "https://store.example.com/p-1?variation=blue" {
		t.Fatalf("results[0].URL = %q, want blue variation URL", results[0].URL)
	}
	if results[0].ImageURL != "https://cdn.example.com/blue.jpg" {
		t.Fatalf("results[0].ImageURL = %q, want blue variation image", results[0].ImageURL)
	}
}

// TestRecommendPinnedPlainProductUsesPreferredVariation verifies pinned products without scoped variation use global preferred variation ordering.
func TestRecommendPinnedPlainProductUsesPreferredVariation(t *testing.T) {
	t.Parallel()

	price := 79.9
	pinnedEntry := port.ProductCatalogEntry{
		ID:           "p-1",
		VariationIDs: []string{"var-blue", "var-red"},
		Datasheets: []port.ProductDatasheetEntry{{
			Realm: "default",
			Name:  "Backpack",
			Price: &price,
			VariationURLs: map[string]string{
				"var-blue": "https://store.example.com/p-1?variation=blue",
				"var-red":  "https://store.example.com/p-1?variation=red",
			},
		}},
	}
	catalog := &catalogStoreStub{
		entriesByID: map[string]port.ProductCatalogEntry{
			"p-1": pinnedEntry,
		},
	}
	svc, err := NewService(affinityStoreStub{}, correlationStoreStub{}, catalog)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	results, err := svc.Recommend(context.Background(), "contact-1", domain.RecommendationQuery{
		BaseTags:           []string{"tier-1"},
		Limit:              1,
		PinnedProductIDs:   []string{"p-1"},
		PreferVariationIDs: []string{"var-red"},
	})
	if err != nil {
		t.Fatalf("Recommend() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].URL != "https://store.example.com/p-1?variation=red" {
		t.Fatalf("results[0].URL = %q, want red variation URL", results[0].URL)
	}
}

// TestRecommendVariationScopedExclusionDoesNotRemoveProduct verifies variation-scoped exclusions remove only variation candidates, not the product itself.
func TestRecommendVariationScopedExclusionDoesNotRemoveProduct(t *testing.T) {
	t.Parallel()

	price := 129.5
	catalog := &catalogStoreStub{
		entries: []port.ProductCatalogEntry{
			{
				ID:           "p-1",
				VariationIDs: []string{"var-red", "var-black"},
				Datasheets: []port.ProductDatasheetEntry{{
					Realm: "default",
					Name:  "Leather Wallet",
					Price: &price,
					VariationURLs: map[string]string{
						"var-red":   "https://store.example.com/p-1?variation=red",
						"var-black": "https://store.example.com/p-1?variation=black",
					},
				}},
			},
		},
	}
	svc, err := NewService(affinityStoreStub{}, correlationStoreStub{}, catalog)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	results, err := svc.Recommend(context.Background(), "contact-1", domain.RecommendationQuery{
		BaseTag:            "cuero",
		Limit:              3,
		ExcludeProductIDs:  []string{"p-1|var-red"},
		PreferVariationIDs: []string{"var-red", "var-black"},
	})
	if err != nil {
		t.Fatalf("Recommend() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].URL != "https://store.example.com/p-1?variation=black" {
		t.Fatalf("results[0].URL = %q, want black variation URL", results[0].URL)
	}
}

// TestRecommendVariationScopedExclusionSteersImageSelection verifies scoped exclusions avoid excluded variation images when alternatives exist.
func TestRecommendVariationScopedExclusionSteersImageSelection(t *testing.T) {
	t.Parallel()

	price := 129.5
	catalog := &catalogStoreStub{
		entries: []port.ProductCatalogEntry{
			{
				ID:           "p-1",
				VariationIDs: []string{"var-red", "var-black"},
				Datasheets: []port.ProductDatasheetEntry{{
					Realm: "default",
					Name:  "Leather Wallet",
					Price: &price,
					VariationURLs: map[string]string{
						"var-red":   "https://store.example.com/p-1?variation=red",
						"var-black": "https://store.example.com/p-1?variation=black",
					},
				}},
				Gallery: []port.ProductGalleryEntry{
					{AssetID: "asset-red", AssetURL: "https://cdn.example.com/red.jpg", VariationIDs: []string{"var-red"}},
					{AssetID: "asset-black", AssetURL: "https://cdn.example.com/black.jpg", VariationIDs: []string{"var-black"}},
				},
			},
		},
	}
	svc, err := NewService(affinityStoreStub{}, correlationStoreStub{}, catalog)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	results, err := svc.Recommend(context.Background(), "contact-1", domain.RecommendationQuery{
		BaseTag:           "cuero",
		Limit:             3,
		ExcludeProductIDs: []string{"p-1|var-red"},
	})
	if err != nil {
		t.Fatalf("Recommend() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].ImageURL != "https://cdn.example.com/black.jpg" {
		t.Fatalf("results[0].ImageURL = %q, want black variation image", results[0].ImageURL)
	}
	if results[0].URL != "https://store.example.com/p-1?variation=black" {
		t.Fatalf("results[0].URL = %q, want black variation URL", results[0].URL)
	}
}

// TestRecommendPlainExcludeRemovesProduct verifies plain product excludes remove candidates completely.
func TestRecommendPlainExcludeRemovesProduct(t *testing.T) {
	t.Parallel()

	price := 129.5
	catalog := &catalogStoreStub{
		entries: []port.ProductCatalogEntry{
			{
				ID: "p-1",
				Datasheets: []port.ProductDatasheetEntry{{
					Realm: "default",
					Name:  "Leather Wallet",
					Price: &price,
					URL:   "https://store.example.com/p-1",
				}},
			},
		},
	}
	svc, err := NewService(affinityStoreStub{}, correlationStoreStub{}, catalog)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	results, err := svc.Recommend(context.Background(), "contact-1", domain.RecommendationQuery{
		BaseTag:           "cuero",
		Limit:             3,
		ExcludeProductIDs: []string{"p-1"},
	})
	if err != nil {
		t.Fatalf("Recommend() error = %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("len(results) = %d, want 0", len(results))
	}
}

// TestRecommendFiltersByDefaultRealmDatasheetPrice verifies min-price filtering uses datasheet realm price, not base product price.
func TestRecommendFiltersByDefaultRealmDatasheetPrice(t *testing.T) {
	t.Parallel()

	price107 := 107000.0
	price144 := 144000.0
	price162 := 162000.0
	catalog := &catalogStoreStub{
		entries: []port.ProductCatalogEntry{
			{
				ID: "p-1",
				Datasheets: []port.ProductDatasheetEntry{{
					Realm: "default",
					Name:  "Totepack Cora Aura",
					Price: &price107,
				}},
			},
			{
				ID: "p-2",
				Datasheets: []port.ProductDatasheetEntry{{
					Realm: "default",
					Name:  "Morral Dream Nubuk",
					Price: &price162,
				}},
			},
			{
				ID: "p-3",
				Datasheets: []port.ProductDatasheetEntry{{
					Realm: "default",
					Name:  "Morral Traveler Terra",
					Price: &price144,
				}},
			},
		},
	}
	svc, err := NewService(affinityStoreStub{}, correlationStoreStub{}, catalog)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	minPrice := 140000.0
	results, err := svc.Recommend(context.Background(), "contact-1", domain.RecommendationQuery{
		BaseTag:    "tier-1",
		CategoryID: "morrales",
		MinPrice:   &minPrice,
		Limit:      4,
	})
	if err != nil {
		t.Fatalf("Recommend() error = %v", err)
	}
	if catalog.lastCategoryIDFilter != "morrales" {
		t.Fatalf("lastCategoryIDFilter = %q, want %q", catalog.lastCategoryIDFilter, "morrales")
	}
	if catalog.lastMinPriceFilter != nil || catalog.lastMaxPriceFilter != nil {
		t.Fatalf("store price filters should be nil to avoid base-price filtering, got min=%v max=%v", catalog.lastMinPriceFilter, catalog.lastMaxPriceFilter)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	if results[0].ID != "p-2" || results[1].ID != "p-3" {
		t.Fatalf("results IDs = [%s, %s], want [p-2, p-3]", results[0].ID, results[1].ID)
	}
}

// TestRecommendExcludePurchasedProducts verifies purchased products are excluded from dynamic and pinned recommendations.
func TestRecommendExcludePurchasedProducts(t *testing.T) {
	t.Parallel()

	price110 := 110000.0
	price160 := 160000.0
	catalog := &catalogStoreStub{
		entriesByID: map[string]port.ProductCatalogEntry{
			"p-1": {
				ID: "p-1",
				Datasheets: []port.ProductDatasheetEntry{{
					Realm: "default",
					Name:  "Purchased Backpack",
					Price: &price160,
				}},
			},
		},
		entries: []port.ProductCatalogEntry{
			{
				ID: "p-1",
				Datasheets: []port.ProductDatasheetEntry{{
					Realm: "default",
					Name:  "Purchased Backpack",
					Price: &price160,
				}},
			},
			{
				ID: "p-2",
				Datasheets: []port.ProductDatasheetEntry{{
					Realm: "default",
					Name:  "Fresh Backpack",
					Price: &price110,
				}},
			},
		},
	}
	svc, err := NewService(
		affinityStorePurchasedStub{purchasedProductIDs: []string{"p-1"}},
		correlationStoreStub{},
		catalog,
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	results, err := svc.Recommend(context.Background(), "contact-1", domain.RecommendationQuery{
		BaseTags:                 []string{"tier-1"},
		Limit:                    3,
		PinnedProductIDs:         []string{"p-1"},
		ExcludePurchasedProducts: true,
	})
	if err != nil {
		t.Fatalf("Recommend() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].ID != "p-2" {
		t.Fatalf("results[0].ID = %q, want %q", results[0].ID, "p-2")
	}
}
