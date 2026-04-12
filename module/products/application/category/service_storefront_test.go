package category_test

import (
	"context"
	"sync/atomic"
	"testing"

	categoryapplication "mannaiah/module/products/application/category"
)

// storefrontNavigationRefresherMock defines refresh behavior for category service tests.
type storefrontNavigationRefresherMock struct {
	// triggerCount tracks refresh triggers.
	triggerCount atomic.Int32
}

// TriggerRefresh records one refresh trigger.
func (m *storefrontNavigationRefresherMock) TriggerRefresh(ctx context.Context) {
	m.triggerCount.Add(1)
}

// TestCreateTriggersStorefrontRefresh verifies successful category creates trigger navigation refreshes.
func TestCreateTriggersStorefrontRefresh(t *testing.T) {
	repo := &mockCategoryRepository{}
	service, err := categoryapplication.NewService(repo)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	refresher := &storefrontNavigationRefresherMock{}
	service.SetStorefrontNavigationRefresher(refresher)

	if _, createErr := service.Create(context.Background(), categoryapplication.CreateCommand{
		Slug: "morrales",
		Name: "Morrales",
	}); createErr != nil {
		t.Fatalf("Create() error = %v", createErr)
	}
	if refresher.triggerCount.Load() != 1 {
		t.Fatalf("triggerCount = %d, want 1", refresher.triggerCount.Load())
	}
}
