package application

import (
	"context"
	"errors"
	"testing"

	analyticsdomain "mannaiah/module/analytics/domain"
	"mannaiah/module/segment/domain"
)

// parentAwareResolverSpy captures filter payloads and returns configurable contact id sets.
type parentAwareResolverSpy struct {
	calls   []analyticsdomain.SegmentFilter
	results map[int][]string
	counts  map[int]int64
}

// ResolveContacts resolves contact ids for analytical filters.
func (r *parentAwareResolverSpy) ResolveContacts(ctx context.Context, filter analyticsdomain.SegmentFilter, page int, limit int) ([]string, error) {
	idx := len(r.calls)
	r.calls = append(r.calls, filter)
	if ids, ok := r.results[idx]; ok {
		return ids, nil
	}
	return []string{}, nil
}

// CountContacts counts contact ids for analytical filters.
func (r *parentAwareResolverSpy) CountContacts(ctx context.Context, filter analyticsdomain.SegmentFilter) (int64, error) {
	idx := len(r.calls)
	r.calls = append(r.calls, filter)
	if count, ok := r.counts[idx]; ok {
		return count, nil
	}
	return 0, nil
}

// TestResolveSubSegmentAppliesParentScope verifies child resolution restricts to parent contacts.
func TestResolveSubSegmentAppliesParentScope(t *testing.T) {
	parentID := "parent-1"
	repo := &stubSegmentRepository{
		getByIDFn: func(ctx context.Context, id string) (*domain.Segment, error) {
			switch id {
			case "parent-1":
				return &domain.Segment{
					ID:      "parent-1",
					Filters: []domain.Filter{{Type: "email_opt_in", Value: true}},
				}, nil
			case "child-1":
				return &domain.Segment{
					ID:              "child-1",
					ParentSegmentID: &parentID,
					Filters:         []domain.Filter{{Type: "first_purchase_only"}},
				}, nil
			}
			return nil, domain.ErrNotFound
		},
	}
	resolver := &parentAwareResolverSpy{
		results: map[int][]string{
			0: {"c-1", "c-2", "c-3"},
			1: {"c-1", "c-3"},
		},
	}

	service, err := NewService(repo, resolver)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	result, err := service.Resolve(context.Background(), "child-1", 1, 1000)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if len(result.ContactIDs) != 2 {
		t.Fatalf("len(ContactIDs) = %d, want 2", len(result.ContactIDs))
	}
	if len(resolver.calls) != 2 {
		t.Fatalf("resolver call count = %d, want 2", len(resolver.calls))
	}
	childFilter := resolver.calls[1]
	if len(childFilter.ContactIDScope) != 3 {
		t.Fatalf("len(ContactIDScope) = %d, want 3", len(childFilter.ContactIDScope))
	}
}

// TestCountSubSegmentAppliesParentScope verifies count restricts to parent contacts.
func TestCountSubSegmentAppliesParentScope(t *testing.T) {
	parentID := "parent-1"
	repo := &stubSegmentRepository{
		getByIDFn: func(ctx context.Context, id string) (*domain.Segment, error) {
			switch id {
			case "parent-1":
				return &domain.Segment{
					ID:      "parent-1",
					Filters: []domain.Filter{{Type: "email_opt_in", Value: true}},
				}, nil
			case "child-1":
				return &domain.Segment{
					ID:              "child-1",
					ParentSegmentID: &parentID,
					Filters:         []domain.Filter{{Type: "first_purchase_only"}},
				}, nil
			}
			return nil, domain.ErrNotFound
		},
	}
	resolver := &parentAwareResolverSpy{
		results: map[int][]string{
			0: {"c-1", "c-2"},
		},
		counts: map[int]int64{
			1: 2,
		},
	}

	service, err := NewService(repo, resolver)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	count, err := service.Count(context.Background(), "child-1")
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("Count() = %d, want 2", count)
	}
	if len(resolver.calls) != 2 {
		t.Fatalf("resolver call count = %d, want 2", len(resolver.calls))
	}
	countFilter := resolver.calls[1]
	if len(countFilter.ContactIDScope) != 2 {
		t.Fatalf("len(ContactIDScope) = %d, want 2", len(countFilter.ContactIDScope))
	}
}

// TestResolveWithoutParentSkipsScope verifies segments without parents do not set scope.
func TestResolveWithoutParentSkipsScope(t *testing.T) {
	repo := &stubSegmentRepository{
		getByIDFn: func(ctx context.Context, id string) (*domain.Segment, error) {
			return &domain.Segment{
				ID:      "seg-1",
				Filters: []domain.Filter{{Type: "email_opt_in", Value: true}},
			}, nil
		},
	}
	resolver := &parentAwareResolverSpy{
		results: map[int][]string{
			0: {"c-1"},
		},
	}

	service, err := NewService(repo, resolver)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, err = service.Resolve(context.Background(), "seg-1", 1, 1000)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if len(resolver.calls) != 1 {
		t.Fatalf("resolver call count = %d, want 1", len(resolver.calls))
	}
	if len(resolver.calls[0].ContactIDScope) != 0 {
		t.Fatalf("ContactIDScope should be empty for non-sub-segments")
	}
}

// TestResolveSubSegmentEmptyParentReturnsEmpty verifies empty parent yields empty child.
func TestResolveSubSegmentEmptyParentReturnsEmpty(t *testing.T) {
	parentID := "parent-empty"
	repo := &stubSegmentRepository{
		getByIDFn: func(ctx context.Context, id string) (*domain.Segment, error) {
			switch id {
			case "parent-empty":
				return &domain.Segment{
					ID:      "parent-empty",
					Filters: []domain.Filter{{Type: "email_opt_in", Value: true}},
				}, nil
			case "child-1":
				return &domain.Segment{
					ID:              "child-1",
					ParentSegmentID: &parentID,
					Filters:         []domain.Filter{{Type: "first_purchase_only"}},
				}, nil
			}
			return nil, domain.ErrNotFound
		},
	}
	resolver := &parentAwareResolverSpy{
		results: map[int][]string{
			0: {},
			1: {},
		},
	}

	service, err := NewService(repo, resolver)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	result, err := service.Resolve(context.Background(), "child-1", 1, 1000)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if len(result.ContactIDs) != 0 {
		t.Fatalf("len(ContactIDs) = %d, want 0", len(result.ContactIDs))
	}
	childFilter := resolver.calls[1]
	if len(childFilter.ContactIDScope) != 1 || childFilter.ContactIDScope[0] != "__impossible__" {
		t.Fatalf("ContactIDScope should contain impossible marker, got %v", childFilter.ContactIDScope)
	}
}

// TestCreateSubSegmentValidatesParent verifies parent existence is checked on create.
func TestCreateSubSegmentValidatesParent(t *testing.T) {
	repo := &stubSegmentRepository{
		getByIDFn: func(ctx context.Context, id string) (*domain.Segment, error) {
			return nil, domain.ErrNotFound
		},
	}
	resolver := &parentAwareResolverSpy{}

	service, err := NewService(repo, resolver)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	parentID := "missing-parent"
	_, err = service.Create(context.Background(), CreateCommand{
		Name:            "child",
		Slug:            "child",
		ParentSegmentID: &parentID,
		Filters:         []domain.Filter{{Type: "email_opt_in", Value: true}},
	})
	if !errors.Is(err, domain.ErrParentNotFound) {
		t.Fatalf("Create() error = %v, want ErrParentNotFound", err)
	}
}

// TestCreateSubSegmentRejectsCircularSelfReference verifies self-referencing parent is rejected.
func TestCreateSubSegmentRejectsCircularSelfReference(t *testing.T) {
	selfID := "seg-1"
	repo := &stubSegmentRepository{
		getByIDFn: func(ctx context.Context, id string) (*domain.Segment, error) {
			return &domain.Segment{ID: "seg-1"}, nil
		},
	}
	resolver := &parentAwareResolverSpy{}

	service, err := NewService(repo, resolver)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, err = service.Update(context.Background(), "seg-1", UpdateCommand{
		ParentSegmentID: &selfID,
	})
	if !errors.Is(err, domain.ErrCircularReference) {
		t.Fatalf("Update() error = %v, want ErrCircularReference", err)
	}
}

// TestCreateSubSegmentRejectsTransitiveCircle verifies A→B→A cycle is rejected.
func TestCreateSubSegmentRejectsTransitiveCircle(t *testing.T) {
	parentA := "seg-a"
	repo := &stubSegmentRepository{
		getByIDFn: func(ctx context.Context, id string) (*domain.Segment, error) {
			switch id {
			case "seg-a":
				return &domain.Segment{ID: "seg-a"}, nil
			case "seg-b":
				return &domain.Segment{ID: "seg-b", ParentSegmentID: &parentA}, nil
			}
			return nil, domain.ErrNotFound
		},
	}
	resolver := &parentAwareResolverSpy{}

	service, err := NewService(repo, resolver)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	parentB := "seg-b"
	_, err = service.Update(context.Background(), "seg-a", UpdateCommand{
		ParentSegmentID: &parentB,
	})
	if !errors.Is(err, domain.ErrCircularReference) {
		t.Fatalf("Update() error = %v, want ErrCircularReference", err)
	}
}

// TestCreateSubSegmentAcceptsValidParent verifies valid parent reference is accepted.
func TestCreateSubSegmentAcceptsValidParent(t *testing.T) {
	repo := &stubSegmentRepository{
		getByIDFn: func(ctx context.Context, id string) (*domain.Segment, error) {
			if id == "parent-1" {
				return &domain.Segment{ID: "parent-1"}, nil
			}
			return nil, domain.ErrNotFound
		},
	}
	resolver := &parentAwareResolverSpy{}

	service, err := NewService(repo, resolver)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	parentID := "parent-1"
	result, err := service.Create(context.Background(), CreateCommand{
		Name:            "child",
		Slug:            "child",
		ParentSegmentID: &parentID,
		Filters:         []domain.Filter{{Type: "email_opt_in", Value: true}},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if result.ParentSegmentID == nil || *result.ParentSegmentID != "parent-1" {
		t.Fatalf("ParentSegmentID = %v, want parent-1", result.ParentSegmentID)
	}
}

// TestUpdateClearsParentSegment verifies empty string clears parent reference.
func TestUpdateClearsParentSegment(t *testing.T) {
	parentID := "parent-1"
	repo := &stubSegmentRepository{
		getByIDFn: func(ctx context.Context, id string) (*domain.Segment, error) {
			return &domain.Segment{
				ID:              "seg-1",
				Name:            "test",
				Slug:            "test",
				ParentSegmentID: &parentID,
				Filters:         []domain.Filter{{Type: "email_opt_in", Value: true}},
			}, nil
		},
	}
	resolver := &parentAwareResolverSpy{}

	service, err := NewService(repo, resolver)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	emptyParent := ""
	result, err := service.Update(context.Background(), "seg-1", UpdateCommand{
		ParentSegmentID: &emptyParent,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if result.ParentSegmentID != nil {
		t.Fatalf("ParentSegmentID = %v, want nil", result.ParentSegmentID)
	}
}
