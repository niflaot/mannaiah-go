package affinity

import (
	"context"
	"errors"
	"testing"

	"mannaiah/module/analytics/domain"
	"mannaiah/module/analytics/port"
)

// mockAffinityStore is a test double for port.AffinityStore.
type mockAffinityStore struct {
	getTagFn       func(ctx context.Context, contactID string, limit int, minScore float64) ([]domain.TagAffinity, error)
	getCategoryFn  func(ctx context.Context, contactID string, limit int, minScore float64) ([]domain.CategoryAffinity, error)
	getVariationFn func(ctx context.Context, contactID string, limit int, minScore float64) ([]domain.VariationAffinity, error)
	getProfileFn   func(ctx context.Context, contactID string, limit int, minScore float64) (*domain.AffinityProfile, error)
	getPurchasedFn func(ctx context.Context, contactID string, limit int) ([]string, error)
	refreshTagFn   func(ctx context.Context) error
	refreshCatFn   func(ctx context.Context) error
	refreshVarFn   func(ctx context.Context) error
}

func (m *mockAffinityStore) GetTagAffinity(ctx context.Context, id string, l int, s float64) ([]domain.TagAffinity, error) {
	return m.getTagFn(ctx, id, l, s)
}
func (m *mockAffinityStore) GetCategoryAffinity(ctx context.Context, id string, l int, s float64) ([]domain.CategoryAffinity, error) {
	return m.getCategoryFn(ctx, id, l, s)
}
func (m *mockAffinityStore) GetVariationAffinity(ctx context.Context, id string, l int, s float64) ([]domain.VariationAffinity, error) {
	return m.getVariationFn(ctx, id, l, s)
}
func (m *mockAffinityStore) GetProfile(ctx context.Context, id string, l int, s float64) (*domain.AffinityProfile, error) {
	return m.getProfileFn(ctx, id, l, s)
}
func (m *mockAffinityStore) GetPurchasedProductIDs(ctx context.Context, id string, l int) ([]string, error) {
	if m.getPurchasedFn == nil {
		return nil, nil
	}
	return m.getPurchasedFn(ctx, id, l)
}
func (m *mockAffinityStore) RefreshTagMV(ctx context.Context) error      { return m.refreshTagFn(ctx) }
func (m *mockAffinityStore) RefreshCategoryMV(ctx context.Context) error { return m.refreshCatFn(ctx) }
func (m *mockAffinityStore) RefreshVariationMV(ctx context.Context) error {
	return m.refreshVarFn(ctx)
}

var _ port.AffinityStore = (*mockAffinityStore)(nil)

// TestNewService_NilDependency verifies constructor validation.
func TestNewService_NilDependency(t *testing.T) {
	if _, err := NewService(nil); !errors.Is(err, ErrNilAffinityStore) {
		t.Errorf("NewService(nil) error = %v, want ErrNilAffinityStore", err)
	}
}

// TestGetTagAffinity_Delegates verifies GetTagAffinity delegation.
func TestGetTagAffinity_Delegates(t *testing.T) {
	store := &mockAffinityStore{
		getTagFn: func(_ context.Context, id string, _ int, _ float64) ([]domain.TagAffinity, error) {
			return []domain.TagAffinity{{ContactID: id, Tag: "coffee"}}, nil
		},
	}
	svc, _ := NewService(store)
	rows, err := svc.GetTagAffinity(context.Background(), "c-1", 10, 0)
	if err != nil {
		t.Fatalf("GetTagAffinity() error = %v", err)
	}
	if len(rows) != 1 || rows[0].Tag != "coffee" {
		t.Errorf("GetTagAffinity() = %v, want [{Tag:coffee}]", rows)
	}
}

// TestGetProfile_Delegates verifies GetProfile delegation.
func TestGetProfile_Delegates(t *testing.T) {
	store := &mockAffinityStore{
		getProfileFn: func(_ context.Context, id string, _ int, _ float64) (*domain.AffinityProfile, error) {
			return &domain.AffinityProfile{ContactID: id}, nil
		},
	}
	svc, _ := NewService(store)
	profile, err := svc.GetProfile(context.Background(), "c-2", 10, 0)
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	if profile.ContactID != "c-2" {
		t.Errorf("GetProfile().ContactID = %q, want %q", profile.ContactID, "c-2")
	}
}

// TestRefreshAll_CallsAllMVs verifies RefreshAll calls all three refresh methods.
func TestRefreshAll_CallsAllMVs(t *testing.T) {
	tagCalled, catCalled, varCalled := false, false, false
	store := &mockAffinityStore{
		refreshTagFn: func(_ context.Context) error { tagCalled = true; return nil },
		refreshCatFn: func(_ context.Context) error { catCalled = true; return nil },
		refreshVarFn: func(_ context.Context) error { varCalled = true; return nil },
	}
	svc, _ := NewService(store)
	if err := svc.RefreshAll(context.Background()); err != nil {
		t.Fatalf("RefreshAll() error = %v", err)
	}
	if !tagCalled || !catCalled || !varCalled {
		t.Errorf("RefreshAll: tag=%v cat=%v var=%v, all must be true", tagCalled, catCalled, varCalled)
	}
}

// TestRefreshAll_StopsOnError verifies RefreshAll stops at the first error.
func TestRefreshAll_StopsOnError(t *testing.T) {
	catCalled := false
	store := &mockAffinityStore{
		refreshTagFn: func(_ context.Context) error { return errors.New("tag mv error") },
		refreshCatFn: func(_ context.Context) error { catCalled = true; return nil },
		refreshVarFn: func(_ context.Context) error { return nil },
	}
	svc, _ := NewService(store)
	err := svc.RefreshAll(context.Background())
	if err == nil {
		t.Fatalf("RefreshAll() expected error, got nil")
	}
	if catCalled {
		t.Errorf("RefreshAll continued after tag error")
	}
}
