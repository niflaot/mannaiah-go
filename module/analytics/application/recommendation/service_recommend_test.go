package recommendation

import (
	"context"
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
	entries []port.ProductCatalogEntry
}

// GetProductsByBaseTags returns predefined test entries.
func (s catalogStoreStub) GetProductsByBaseTags(_ context.Context, _ []string, _ string, _ []string, _ string, _ []string, _ []string, _ int) ([]port.ProductCatalogEntry, error) {
	return s.entries, nil
}

// GetProductsByIDs returns no entries for this test.
func (catalogStoreStub) GetProductsByIDs(_ context.Context, _ []string, _ []string) ([]port.ProductCatalogEntry, error) {
	return nil, nil
}

// TestRecommendIncludesProductsWithoutImage verifies products are not dropped when image URLs are unavailable.
func TestRecommendIncludesProductsWithoutImage(t *testing.T) {
	t.Parallel()

	price := 129.5
	svc, err := NewService(
		affinityStoreStub{},
		correlationStoreStub{},
		catalogStoreStub{entries: []port.ProductCatalogEntry{
			{
				ID: "p-1",
				Datasheets: []port.ProductDatasheetEntry{{
					Realm: "default",
					Name:  "Leather Wallet",
					Price: &price,
				}},
				Gallery: nil,
			},
		}},
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
