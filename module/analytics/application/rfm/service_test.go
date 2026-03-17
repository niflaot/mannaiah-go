package rfm

import (
	"context"
	"errors"
	"testing"

	"mannaiah/module/analytics/domain"
	"mannaiah/module/analytics/port"
)

// mockRFMStore is a test double for port.RFMStore.
type mockRFMStore struct {
	scoreContactFn func(ctx context.Context, contactID string, bands []domain.RFMBandConfig) (*domain.RFMScore, error)
	scoreBatchFn   func(ctx context.Context, contactIDs []string, bands []domain.RFMBandConfig) ([]domain.RFMScore, error)
	refreshMVFn    func(ctx context.Context) error
	percentilesFn  func(ctx context.Context) ([4]float64, error)
}

func (m *mockRFMStore) ScoreContact(ctx context.Context, id string, bands []domain.RFMBandConfig) (*domain.RFMScore, error) {
	return m.scoreContactFn(ctx, id, bands)
}

func (m *mockRFMStore) ScoreBatch(ctx context.Context, ids []string, bands []domain.RFMBandConfig) ([]domain.RFMScore, error) {
	return m.scoreBatchFn(ctx, ids, bands)
}

func (m *mockRFMStore) RefreshMV(ctx context.Context) error {
	return m.refreshMVFn(ctx)
}

func (m *mockRFMStore) ComputeMonetaryPercentiles(ctx context.Context) ([4]float64, error) {
	return m.percentilesFn(ctx)
}

// mockGroupRepo is a test double for port.RFMGroupRepository.
type mockGroupRepo struct {
	createFn         func(ctx context.Context, group *domain.RFMGroup) error
	getByIDFn        func(ctx context.Context, id string) (*domain.RFMGroup, error)
	getBySlugFn      func(ctx context.Context, slug string) (*domain.RFMGroup, error)
	listFn           func(ctx context.Context) ([]domain.RFMGroup, error)
	updateFn         func(ctx context.Context, group *domain.RFMGroup) error
	deleteFn         func(ctx context.Context, id string) error
	getBandConfigsFn func(ctx context.Context) ([]domain.RFMBandConfig, error)
	updateBandFn     func(ctx context.Context, cfg domain.RFMBandConfig) error
	seedDefaultsFn   func(ctx context.Context) error
}

func (m *mockGroupRepo) Create(ctx context.Context, g *domain.RFMGroup) error {
	return m.createFn(ctx, g)
}
func (m *mockGroupRepo) GetByID(ctx context.Context, id string) (*domain.RFMGroup, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockGroupRepo) GetBySlug(ctx context.Context, slug string) (*domain.RFMGroup, error) {
	return m.getBySlugFn(ctx, slug)
}
func (m *mockGroupRepo) List(ctx context.Context) ([]domain.RFMGroup, error) {
	return m.listFn(ctx)
}
func (m *mockGroupRepo) Update(ctx context.Context, g *domain.RFMGroup) error {
	return m.updateFn(ctx, g)
}
func (m *mockGroupRepo) Delete(ctx context.Context, id string) error {
	return m.deleteFn(ctx, id)
}
func (m *mockGroupRepo) GetBandConfigs(ctx context.Context) ([]domain.RFMBandConfig, error) {
	return m.getBandConfigsFn(ctx)
}
func (m *mockGroupRepo) UpdateBandConfig(ctx context.Context, cfg domain.RFMBandConfig) error {
	return m.updateBandFn(ctx, cfg)
}
func (m *mockGroupRepo) SeedDefaultBands(ctx context.Context) error {
	return m.seedDefaultsFn(ctx)
}

var _ port.RFMGroupRepository = (*mockGroupRepo)(nil)
var _ port.RFMStore = (*mockRFMStore)(nil)

func newTestService(t *testing.T) (*RFMService, *mockRFMStore, *mockGroupRepo) {
	t.Helper()
	store := &mockRFMStore{}
	repo := &mockGroupRepo{}
	svc, err := NewService(store, repo)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	return svc, store, repo
}

// TestNewService_NilDependencies verifies constructor validation.
func TestNewService_NilDependencies(t *testing.T) {
	if _, err := NewService(nil, &mockGroupRepo{}); !errors.Is(err, ErrNilRFMStore) {
		t.Errorf("NewService(nil store) error = %v, want ErrNilRFMStore", err)
	}
	if _, err := NewService(&mockRFMStore{}, nil); !errors.Is(err, ErrNilGroupRepo) {
		t.Errorf("NewService(nil repo) error = %v, want ErrNilGroupRepo", err)
	}
}

// TestCreateGroup verifies group creation delegation.
func TestCreateGroup(t *testing.T) {
	svc, _, repo := newTestService(t)
	repo.createFn = func(_ context.Context, g *domain.RFMGroup) error {
		g.ID = "id-1"
		return nil
	}

	got, err := svc.CreateGroup(context.Background(), domain.RFMGroup{Name: "Champions", Slug: "champions"})
	if err != nil {
		t.Fatalf("CreateGroup() error = %v", err)
	}
	if got.ID != "id-1" {
		t.Errorf("CreateGroup().ID = %q, want %q", got.ID, "id-1")
	}
}

// TestListGroups verifies group list delegation.
func TestListGroups(t *testing.T) {
	svc, _, repo := newTestService(t)
	repo.listFn = func(_ context.Context) ([]domain.RFMGroup, error) {
		return []domain.RFMGroup{{ID: "id-1"}, {ID: "id-2"}}, nil
	}

	groups, err := svc.ListGroups(context.Background())
	if err != nil {
		t.Fatalf("ListGroups() error = %v", err)
	}
	if len(groups) != 2 {
		t.Errorf("ListGroups() len = %d, want 2", len(groups))
	}
}

// TestGetBands_Cache verifies band cache behaviour (second call must not hit repo).
func TestGetBands_Cache(t *testing.T) {
	svc, _, repo := newTestService(t)
	callCount := 0
	repo.getBandConfigsFn = func(_ context.Context) ([]domain.RFMBandConfig, error) {
		callCount++
		return []domain.RFMBandConfig{{Dimension: domain.DimensionRecency}}, nil
	}

	_, _ = svc.GetBands(context.Background())
	_, _ = svc.GetBands(context.Background())
	if callCount != 1 {
		t.Errorf("repo called %d times, want 1 (cache hit on second call)", callCount)
	}
}

// TestUpdateBand_InvalidatesCache verifies that UpdateBand clears the cache.
func TestUpdateBand_InvalidatesCache(t *testing.T) {
	svc, _, repo := newTestService(t)
	callCount := 0
	repo.getBandConfigsFn = func(_ context.Context) ([]domain.RFMBandConfig, error) {
		callCount++
		return []domain.RFMBandConfig{{Dimension: domain.DimensionRecency}}, nil
	}
	repo.updateBandFn = func(_ context.Context, _ domain.RFMBandConfig) error { return nil }

	_, _ = svc.GetBands(context.Background())
	_ = svc.UpdateBand(context.Background(), domain.RFMBandConfig{})
	_, _ = svc.GetBands(context.Background())
	if callCount != 2 {
		t.Errorf("repo called %d times after invalidation, want 2", callCount)
	}
}

// TestScoreContact_DelegatesToStore verifies ScoreContact delegation.
func TestScoreContact_DelegatesToStore(t *testing.T) {
	svc, store, repo := newTestService(t)
	repo.getBandConfigsFn = func(_ context.Context) ([]domain.RFMBandConfig, error) {
		return []domain.RFMBandConfig{}, nil
	}
	store.scoreContactFn = func(_ context.Context, id string, _ []domain.RFMBandConfig) (*domain.RFMScore, error) {
		return &domain.RFMScore{ContactID: id, RScore: 5}, nil
	}

	score, err := svc.ScoreContact(context.Background(), "c-1")
	if err != nil {
		t.Fatalf("ScoreContact() error = %v", err)
	}
	if score.RScore != 5 {
		t.Errorf("ScoreContact().RScore = %d, want 5", score.RScore)
	}
}

// TestRefreshMV_DelegatesToStore verifies RefreshMV delegation.
func TestRefreshMV_DelegatesToStore(t *testing.T) {
	svc, store, _ := newTestService(t)
	called := false
	store.refreshMVFn = func(_ context.Context) error {
		called = true
		return nil
	}
	if err := svc.RefreshMV(context.Background()); err != nil {
		t.Fatalf("RefreshMV() error = %v", err)
	}
	if !called {
		t.Errorf("RefreshMV did not delegate to store")
	}
}
