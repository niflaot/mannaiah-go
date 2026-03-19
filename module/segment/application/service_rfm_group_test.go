package application

import (
	"context"
	"errors"
	"testing"

	analyticsdomain "mannaiah/module/analytics/domain"
	"mannaiah/module/segment/domain"
	segmentport "mannaiah/module/segment/port"
)

// stubSegmentRepository defines configurable segment repository behavior for tests.
type stubSegmentRepository struct {
	getByIDFn func(ctx context.Context, id string) (*domain.Segment, error)
}

// Create persists segment rows.
func (s stubSegmentRepository) Create(ctx context.Context, segment *domain.Segment) error {
	return nil
}

// GetByID retrieves one segment by id.
func (s stubSegmentRepository) GetByID(ctx context.Context, id string) (*domain.Segment, error) {
	if s.getByIDFn != nil {
		return s.getByIDFn(ctx, id)
	}

	return nil, domain.ErrNotFound
}

// List retrieves segment rows.
func (s stubSegmentRepository) List(ctx context.Context, page int, limit int) ([]domain.Segment, int64, error) {
	return []domain.Segment{}, 0, nil
}

// Update persists segment row updates.
func (s stubSegmentRepository) Update(ctx context.Context, segment *domain.Segment) error {
	return nil
}

// Delete removes one segment by id.
func (s stubSegmentRepository) Delete(ctx context.Context, id string) error {
	return nil
}

var _ segmentport.Repository = (*stubSegmentRepository)(nil)

// resolverSpy captures analytics-filter payloads observed during resolve/count.
type resolverSpy struct {
	lastFilter analyticsdomain.SegmentFilter
	countCalls int
	countValue int64
}

// ResolveContacts resolves contact ids for analytical filters.
func (r *resolverSpy) ResolveContacts(ctx context.Context, filter analyticsdomain.SegmentFilter, page int, limit int) ([]string, error) {
	r.lastFilter = filter
	return []string{}, nil
}

// CountContacts counts contact ids for analytical filters.
func (r *resolverSpy) CountContacts(ctx context.Context, filter analyticsdomain.SegmentFilter) (int64, error) {
	r.lastFilter = filter
	r.countCalls++

	return r.countValue, nil
}

// stubRFMGroupRepository defines configurable RFM-group repository behavior for tests.
type stubRFMGroupRepository struct {
	listFn func(ctx context.Context) ([]analyticsdomain.RFMGroup, error)
}

// Create persists a new RFM group.
func (s stubRFMGroupRepository) Create(ctx context.Context, group *analyticsdomain.RFMGroup) error {
	return nil
}

// GetByID retrieves one RFM group by identifier.
func (s stubRFMGroupRepository) GetByID(ctx context.Context, id string) (*analyticsdomain.RFMGroup, error) {
	return nil, errors.New("not implemented")
}

// GetBySlug retrieves one RFM group by slug.
func (s stubRFMGroupRepository) GetBySlug(ctx context.Context, slug string) (*analyticsdomain.RFMGroup, error) {
	return nil, errors.New("not implemented")
}

// List retrieves all RFM groups.
func (s stubRFMGroupRepository) List(ctx context.Context) ([]analyticsdomain.RFMGroup, error) {
	if s.listFn != nil {
		return s.listFn(ctx)
	}

	return []analyticsdomain.RFMGroup{}, nil
}

// Update persists RFM group updates.
func (s stubRFMGroupRepository) Update(ctx context.Context, group *analyticsdomain.RFMGroup) error {
	return nil
}

// Delete removes one RFM group by identifier.
func (s stubRFMGroupRepository) Delete(ctx context.Context, id string) error {
	return nil
}

// GetBandConfigs retrieves all RFM band threshold configurations.
func (s stubRFMGroupRepository) GetBandConfigs(ctx context.Context) ([]analyticsdomain.RFMBandConfig, error) {
	return []analyticsdomain.RFMBandConfig{}, nil
}

// UpdateBandConfig persists a single RFM band configuration.
func (s stubRFMGroupRepository) UpdateBandConfig(ctx context.Context, cfg analyticsdomain.RFMBandConfig) error {
	return nil
}

// SeedDefaultBands creates default R/F/M band configs when none exist.
func (s stubRFMGroupRepository) SeedDefaultBands(ctx context.Context) error {
	return nil
}

// intPtr returns an int pointer.
func intPtr(value int) *int {
	return &value
}

// TestCountExpandsRFMGroupClause verifies rfm_group clauses are expanded to rfm_range before resolver count.
func TestCountExpandsRFMGroupClause(t *testing.T) {
	repository := stubSegmentRepository{
		getByIDFn: func(ctx context.Context, id string) (*domain.Segment, error) {
			return &domain.Segment{
				ID: "seg-1",
				Filters: []domain.Filter{
					{Type: "rfm_group", Parameters: map[string]any{"slug": "champions"}},
				},
			}, nil
		},
	}
	resolver := &resolverSpy{countValue: 7}
	rfmGroups := stubRFMGroupRepository{
		listFn: func(ctx context.Context) ([]analyticsdomain.RFMGroup, error) {
			return []analyticsdomain.RFMGroup{
				{
					ID:   "g-1",
					Slug: "champions",
					Conditions: analyticsdomain.RFMGroupConditions{
						RMin: intPtr(4),
						FMin: intPtr(4),
						MMin: intPtr(4),
					},
				},
			}, nil
		},
	}

	service, err := NewService(repository, resolver, rfmGroups)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	count, err := service.Count(context.Background(), "seg-1")
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 7 {
		t.Fatalf("Count() = %d, want 7", count)
	}
	if resolver.countCalls != 1 {
		t.Fatalf("resolver.CountContacts calls = %d, want 1", resolver.countCalls)
	}
	if len(resolver.lastFilter.Clauses) != 1 {
		t.Fatalf("len(resolver.lastFilter.Clauses) = %d, want 1", len(resolver.lastFilter.Clauses))
	}
	clause := resolver.lastFilter.Clauses[0]
	if clause.Type != "rfm_range" {
		t.Fatalf("clause.Type = %q, want rfm_range", clause.Type)
	}
	if clause.Exclude {
		t.Fatalf("clause.Exclude = true, want false")
	}
	if clause.Parameters["rMin"] != 4 || clause.Parameters["fMin"] != 4 || clause.Parameters["mMin"] != 4 {
		t.Fatalf("clause.Parameters = %#v, want rMin/fMin/mMin = 4", clause.Parameters)
	}
}

// TestCountExpandsExcludedEmptyRFMGroup verifies exclude + empty group conditions resolves to NOT(true).
func TestCountExpandsExcludedEmptyRFMGroup(t *testing.T) {
	repository := stubSegmentRepository{
		getByIDFn: func(ctx context.Context, id string) (*domain.Segment, error) {
			return &domain.Segment{
				ID: "seg-2",
				Filters: []domain.Filter{
					{Type: "rfm_group", Exclude: true, Parameters: map[string]any{"slug": "all-rfm"}},
				},
			}, nil
		},
	}
	resolver := &resolverSpy{countValue: 0}
	rfmGroups := stubRFMGroupRepository{
		listFn: func(ctx context.Context) ([]analyticsdomain.RFMGroup, error) {
			return []analyticsdomain.RFMGroup{
				{
					ID:         "g-2",
					Slug:       "all-rfm",
					Conditions: analyticsdomain.RFMGroupConditions{},
				},
			}, nil
		},
	}

	service, err := NewService(repository, resolver, rfmGroups)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, err = service.Count(context.Background(), "seg-2")
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if len(resolver.lastFilter.Clauses) != 1 {
		t.Fatalf("len(resolver.lastFilter.Clauses) = %d, want 1", len(resolver.lastFilter.Clauses))
	}
	clause := resolver.lastFilter.Clauses[0]
	if clause.Type != "__always_true__" {
		t.Fatalf("clause.Type = %q, want __always_true__", clause.Type)
	}
	if !clause.Exclude {
		t.Fatalf("clause.Exclude = false, want true")
	}
}

// TestPreviewCountRejectsUnknownRFMGroup verifies unknown group slug returns invalid-filter errors.
func TestPreviewCountRejectsUnknownRFMGroup(t *testing.T) {
	repository := stubSegmentRepository{}
	resolver := &resolverSpy{countValue: 1}
	rfmGroups := stubRFMGroupRepository{
		listFn: func(ctx context.Context) ([]analyticsdomain.RFMGroup, error) {
			return []analyticsdomain.RFMGroup{}, nil
		},
	}

	service, err := NewService(repository, resolver, rfmGroups)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, err = service.PreviewCount(context.Background(), []domain.Filter{
		{Type: "rfm_group", Parameters: map[string]any{"slug": "missing"}},
	})
	if !errors.Is(err, domain.ErrInvalidFilter) {
		t.Fatalf("PreviewCount() error = %v, want ErrInvalidFilter", err)
	}
	if resolver.countCalls != 0 {
		t.Fatalf("resolver.CountContacts calls = %d, want 0", resolver.countCalls)
	}
}

// TestPreviewCountRejectsRFMGroupWhenRepositoryUnavailable verifies missing lookup dependency handling.
func TestPreviewCountRejectsRFMGroupWhenRepositoryUnavailable(t *testing.T) {
	repository := stubSegmentRepository{}
	resolver := &resolverSpy{countValue: 1}

	service, err := NewService(repository, resolver)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, err = service.PreviewCount(context.Background(), []domain.Filter{
		{Type: "rfm_group", Parameters: map[string]any{"slug": "champions"}},
	})
	if !errors.Is(err, ErrRFMGroupRepositoryUnavailable) {
		t.Fatalf("PreviewCount() error = %v, want ErrRFMGroupRepositoryUnavailable", err)
	}
	if resolver.countCalls != 0 {
		t.Fatalf("resolver.CountContacts calls = %d, want 0", resolver.countCalls)
	}
}
