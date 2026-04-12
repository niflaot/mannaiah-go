package product

import (
	"context"
	"sync/atomic"
	"testing"

	productdomain "mannaiah/module/products/domain/product"
)

// storefrontNavigationRefresherMock defines refresh behavior for product service tests.
type storefrontNavigationRefresherMock struct {
	// triggerCount tracks refresh triggers.
	triggerCount atomic.Int32
}

// TriggerRefresh records one refresh trigger.
func (m *storefrontNavigationRefresherMock) TriggerRefresh(ctx context.Context) {
	m.triggerCount.Add(1)
}

// TestCreateTriggersStorefrontRefresh verifies successful creates trigger navigation refreshes.
func TestCreateTriggersStorefrontRefresh(t *testing.T) {
	refresher := &storefrontNavigationRefresherMock{}
	service, err := NewService(repositoryMock{
		createFn: func(ctx context.Context, product *productdomain.Product) error {
			product.ID = "p-1"
			return nil
		},
		getFn:    func(ctx context.Context, id string) (*productdomain.Product, error) { return nil, nil },
		listFn:   func(ctx context.Context) ([]productdomain.Product, error) { return nil, nil },
		updateFn: func(ctx context.Context, product *productdomain.Product) error { return nil },
		deleteFn: func(ctx context.Context, id string) error { return nil },
	}, assetLookupMock{existsFn: func(ctx context.Context, id string) (bool, error) { return true, nil }})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	service.SetStorefrontNavigationRefresher(refresher)

	if _, createErr := service.Create(context.Background(), CreateCommand{SKU: "SKU-1"}); createErr != nil {
		t.Fatalf("Create() error = %v", createErr)
	}
	if refresher.triggerCount.Load() != 1 {
		t.Fatalf("triggerCount = %d, want 1", refresher.triggerCount.Load())
	}
}
