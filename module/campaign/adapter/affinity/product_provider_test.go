package affinity

import (
	"context"
	"testing"

	recommendationapp "mannaiah/module/analytics/application/recommendation"
	analyticsdomain "mannaiah/module/analytics/domain"
	analyticsport "mannaiah/module/analytics/port"
	campaigndomain "mannaiah/module/campaign/domain"
)

type affinityStoreTestStub struct{}

// GetTagAffinity returns empty affinity rows.
func (affinityStoreTestStub) GetTagAffinity(_ context.Context, _ string, _ int, _ float64) ([]analyticsdomain.TagAffinity, error) {
	return nil, nil
}

// GetCategoryAffinity returns empty affinity rows.
func (affinityStoreTestStub) GetCategoryAffinity(_ context.Context, _ string, _ int, _ float64) ([]analyticsdomain.CategoryAffinity, error) {
	return nil, nil
}

// GetVariationAffinity returns empty affinity rows.
func (affinityStoreTestStub) GetVariationAffinity(_ context.Context, _ string, _ int, _ float64) ([]analyticsdomain.VariationAffinity, error) {
	return nil, nil
}

// GetProfile returns an empty affinity profile.
func (affinityStoreTestStub) GetProfile(_ context.Context, _ string, _ int, _ float64) (*analyticsdomain.AffinityProfile, error) {
	return &analyticsdomain.AffinityProfile{}, nil
}

// GetPurchasedProductIDs returns no purchased product IDs.
func (affinityStoreTestStub) GetPurchasedProductIDs(_ context.Context, _ string, _ int) ([]string, error) {
	return nil, nil
}

// RefreshTagMV is a no-op for tests.
func (affinityStoreTestStub) RefreshTagMV(_ context.Context) error { return nil }

// RefreshCategoryMV is a no-op for tests.
func (affinityStoreTestStub) RefreshCategoryMV(_ context.Context) error { return nil }

// RefreshVariationMV is a no-op for tests.
func (affinityStoreTestStub) RefreshVariationMV(_ context.Context) error { return nil }

type correlationStoreTestStub struct{}

// GetCorrelations returns no correlations.
func (correlationStoreTestStub) GetCorrelations(_ context.Context, _ []string) ([]analyticsport.TagCorrelation, error) {
	return nil, nil
}

type catalogStoreTestStub struct {
	entries []analyticsport.ProductCatalogEntry
}

// GetProductsByBaseTags returns predefined test entries.
func (s catalogStoreTestStub) GetProductsByBaseTags(
	_ context.Context,
	_ []string,
	_ string,
	_ []string,
	_ string,
	_ []string,
	_ []string,
	_ []string,
	_ []string,
	_ *float64,
	_ *float64,
	_ []string,
	_ []string,
	_ int,
) ([]analyticsport.ProductCatalogEntry, error) {
	return s.entries, nil
}

// GetProductsByIDs returns no entries for this test.
func (catalogStoreTestStub) GetProductsByIDs(_ context.Context, _ []string, _ []string) ([]analyticsport.ProductCatalogEntry, error) {
	return nil, nil
}

// TestGetProductsForcesDefaultRealm verifies marketing recommendations always resolve using the default realm.
func TestGetProductsForcesDefaultRealm(t *testing.T) {
	t.Parallel()

	defaultPrice := 49.9
	wooPrice := 79.9
	recommendationService, err := recommendationapp.NewService(
		affinityStoreTestStub{},
		correlationStoreTestStub{},
		catalogStoreTestStub{entries: []analyticsport.ProductCatalogEntry{
			{
				ID: "product-1",
				Datasheets: []analyticsport.ProductDatasheetEntry{
					{Realm: "default", Name: "Default Name", Price: &defaultPrice, URL: "https://default.example.com/p/1"},
					{Realm: "woo", Name: "Woo Name", Price: &wooPrice, URL: "https://woo.example.com/p/1"},
				},
			},
		}},
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	provider, err := NewProductProvider(recommendationService)
	if err != nil {
		t.Fatalf("NewProductProvider() error = %v", err)
	}

	products, err := provider.GetProducts(context.Background(), "campaign-1", "contact-1", campaigndomain.ProductBlock{
		ID:       "hero_products",
		BaseTags: []string{"cuero"},
		Realm:    "woo",
		Limit:    1,
	})
	if err != nil {
		t.Fatalf("GetProducts() error = %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("len(products) = %d, want 1", len(products))
	}
	if products[0].Name != "Default Name" {
		t.Fatalf("products[0].Name = %q, want %q", products[0].Name, "Default Name")
	}
	if products[0].Price != defaultPrice {
		t.Fatalf("products[0].Price = %v, want %v", products[0].Price, defaultPrice)
	}
	if products[0].URL != "https://default.example.com/p/1" {
		t.Fatalf("products[0].URL = %q, want %q", products[0].URL, "https://default.example.com/p/1")
	}
}
