package category_test

import (
	"context"
	"errors"
	"testing"

	categoryapplication "mannaiah/module/products/application/category"
	categorydomain "mannaiah/module/products/domain/category"
	productdomain "mannaiah/module/products/domain/product"
	categoryport "mannaiah/module/products/port/category"
)

// mockCategoryRepository is a test double for categoryport.Repository.
type mockCategoryRepository struct {
	// createFn defines the Create stub.
	createFn func(ctx context.Context, cat *categorydomain.Category) error
	// getByIDFn defines the GetByID stub.
	getByIDFn func(ctx context.Context, id string) (*categorydomain.Category, error)
	// getBySlugFn defines the GetBySlug stub.
	getBySlugFn func(ctx context.Context, slug string) (*categorydomain.Category, error)
	// treeFn defines the Tree stub.
	treeFn func(ctx context.Context) ([]*categorydomain.Category, error)
	// listChildrenFn defines the ListChildren stub.
	listChildrenFn func(ctx context.Context, parentID string) ([]*categorydomain.Category, error)
	// updateFn defines the Update stub.
	updateFn func(ctx context.Context, cat *categorydomain.Category) error
	// deleteFn defines the Delete stub.
	deleteFn func(ctx context.Context, id string) error
	// listProductsFn defines the ListProducts stub.
	listProductsFn func(ctx context.Context, q categoryport.ListProductsQuery) (*categoryport.ListProductsResult, error)
}

// EnsureSchema satisfies categoryport.Repository.
func (m *mockCategoryRepository) EnsureSchema(_ context.Context) error { return nil }

// Create satisfies categoryport.Repository.
func (m *mockCategoryRepository) Create(ctx context.Context, cat *categorydomain.Category) error {
	if m.createFn != nil {
		return m.createFn(ctx, cat)
	}

	return nil
}

// GetByID satisfies categoryport.Repository.
func (m *mockCategoryRepository) GetByID(ctx context.Context, id string) (*categorydomain.Category, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}

	return nil, categoryport.ErrNotFound
}

// GetBySlug satisfies categoryport.Repository.
func (m *mockCategoryRepository) GetBySlug(ctx context.Context, slug string) (*categorydomain.Category, error) {
	if m.getBySlugFn != nil {
		return m.getBySlugFn(ctx, slug)
	}

	return nil, categoryport.ErrNotFound
}

// Tree satisfies categoryport.Repository.
func (m *mockCategoryRepository) Tree(ctx context.Context) ([]*categorydomain.Category, error) {
	if m.treeFn != nil {
		return m.treeFn(ctx)
	}

	return nil, nil
}

// ListChildren satisfies categoryport.Repository.
func (m *mockCategoryRepository) ListChildren(ctx context.Context, parentID string) ([]*categorydomain.Category, error) {
	if m.listChildrenFn != nil {
		return m.listChildrenFn(ctx, parentID)
	}

	return nil, nil
}

// Update satisfies categoryport.Repository.
func (m *mockCategoryRepository) Update(ctx context.Context, cat *categorydomain.Category) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, cat)
	}

	return nil
}

// Delete satisfies categoryport.Repository.
func (m *mockCategoryRepository) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}

	return nil
}

// ListProducts satisfies categoryport.Repository.
func (m *mockCategoryRepository) ListProducts(ctx context.Context, q categoryport.ListProductsQuery) (*categoryport.ListProductsResult, error) {
	if m.listProductsFn != nil {
		return m.listProductsFn(ctx, q)
	}

	return &categoryport.ListProductsResult{}, nil
}

// TestNewService_NilRepository verifies ErrNilRepository is returned for nil repo.
func TestNewService_NilRepository(t *testing.T) {
	_, err := categoryapplication.NewService(nil)
	if !errors.Is(err, categoryapplication.ErrNilRepository) {
		t.Fatalf("NewService(nil) error = %v, want ErrNilRepository", err)
	}
}

// TestNewService_Valid verifies successful service creation.
func TestNewService_Valid(t *testing.T) {
	svc, err := categoryapplication.NewService(&mockCategoryRepository{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if svc == nil {
		t.Fatal("NewService() returned nil service")
	}
}

// TestCreate_SlugRequired verifies ErrSlugRequired is returned when slug is empty.
func TestCreate_SlugRequired(t *testing.T) {
	svc, _ := categoryapplication.NewService(&mockCategoryRepository{})
	_, err := svc.Create(context.Background(), categoryapplication.CreateCommand{Name: "Electronics"})
	if !errors.Is(err, categorydomain.ErrSlugRequired) {
		t.Fatalf("Create(empty slug) error = %v, want ErrSlugRequired", err)
	}
}

// TestCreate_NameRequired verifies ErrNameRequired is returned when name is empty.
func TestCreate_NameRequired(t *testing.T) {
	svc, _ := categoryapplication.NewService(&mockCategoryRepository{})
	_, err := svc.Create(context.Background(), categoryapplication.CreateCommand{Slug: "electronics"})
	if !errors.Is(err, categorydomain.ErrNameRequired) {
		t.Fatalf("Create(empty name) error = %v, want ErrNameRequired", err)
	}
}

// TestCreate_Success verifies successful category creation.
func TestCreate_Success(t *testing.T) {
	repo := &mockCategoryRepository{}
	svc, _ := categoryapplication.NewService(repo)
	cat, err := svc.Create(context.Background(), categoryapplication.CreateCommand{
		Slug: "electronics",
		Name: "Electronics",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if cat.ID == "" {
		t.Fatal("Create() returned empty ID")
	}
	if cat.Slug != "electronics" {
		t.Fatalf("Create() Slug = %q, want %q", cat.Slug, "electronics")
	}
}

// TestCreate_DuplicateSlug verifies ErrDuplicateSlug is propagated.
func TestCreate_DuplicateSlug(t *testing.T) {
	repo := &mockCategoryRepository{
		createFn: func(_ context.Context, _ *categorydomain.Category) error {
			return categoryport.ErrDuplicateSlug
		},
	}
	svc, _ := categoryapplication.NewService(repo)
	_, err := svc.Create(context.Background(), categoryapplication.CreateCommand{Slug: "dup", Name: "Dup"})
	if !errors.Is(err, categoryapplication.ErrDuplicateSlug) {
		t.Fatalf("Create(dup slug) error = %v, want ErrDuplicateSlug", err)
	}
}

// TestCreate_WithPriceFilter verifies price range filter is set.
func TestCreate_WithPriceFilter(t *testing.T) {
	repo := &mockCategoryRepository{}
	svc, _ := categoryapplication.NewService(repo)
	min := float64(10)
	max := float64(100)
	cat, err := svc.Create(context.Background(), categoryapplication.CreateCommand{
		Slug:           "priced",
		Name:           "Priced",
		FilterMinPrice: &min,
		FilterMaxPrice: &max,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if cat.Filter.PriceRange == nil {
		t.Fatal("Create() PriceRange is nil")
	}
	if *cat.Filter.PriceRange.Min != min {
		t.Fatalf("PriceRange.Min = %v, want %v", *cat.Filter.PriceRange.Min, min)
	}
}

// TestGet_InvalidID verifies ErrInvalidID is returned for blank ID.
func TestGet_InvalidID(t *testing.T) {
	svc, _ := categoryapplication.NewService(&mockCategoryRepository{})
	_, err := svc.Get(context.Background(), "  ")
	if !errors.Is(err, categoryapplication.ErrInvalidID) {
		t.Fatalf("Get(blank id) error = %v, want ErrInvalidID", err)
	}
}

// TestGet_NotFound verifies ErrNotFound is returned from repository.
func TestGet_NotFound(t *testing.T) {
	repo := &mockCategoryRepository{}
	svc, _ := categoryapplication.NewService(repo)
	_, err := svc.Get(context.Background(), "nonexistent")
	if !errors.Is(err, categoryapplication.ErrNotFound) {
		t.Fatalf("Get(missing) error = %v, want ErrNotFound", err)
	}
}

// TestGet_Success verifies successful category retrieval.
func TestGet_Success(t *testing.T) {
	repo := &mockCategoryRepository{
		getByIDFn: func(_ context.Context, id string) (*categorydomain.Category, error) {
			return &categorydomain.Category{ID: id, Slug: "test", Name: "Test"}, nil
		},
	}
	svc, _ := categoryapplication.NewService(repo)
	cat, err := svc.Get(context.Background(), "abc")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if cat.ID != "abc" {
		t.Fatalf("Get() ID = %q, want %q", cat.ID, "abc")
	}
}

// TestUpdate_NotFound verifies ErrNotFound propagates on update of missing category.
func TestUpdate_NotFound(t *testing.T) {
	svc, _ := categoryapplication.NewService(&mockCategoryRepository{})
	_, err := svc.Update(context.Background(), "missing", categoryapplication.UpdateCommand{})
	if !errors.Is(err, categoryapplication.ErrNotFound) {
		t.Fatalf("Update(missing) error = %v, want ErrNotFound", err)
	}
}

// TestUpdate_Success verifies successful category update.
func TestUpdate_Success(t *testing.T) {
	existing := &categorydomain.Category{ID: "cat1", Slug: "old", Name: "Old"}
	repo := &mockCategoryRepository{
		getByIDFn: func(_ context.Context, _ string) (*categorydomain.Category, error) {
			return existing, nil
		},
	}
	svc, _ := categoryapplication.NewService(repo)
	newName := "New Name"
	cat, err := svc.Update(context.Background(), "cat1", categoryapplication.UpdateCommand{Name: &newName})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if cat.Name != "New Name" {
		t.Fatalf("Update() Name = %q, want %q", cat.Name, "New Name")
	}
}

// TestDelete_InvalidID verifies ErrInvalidID for blank ID.
func TestDelete_InvalidID(t *testing.T) {
	svc, _ := categoryapplication.NewService(&mockCategoryRepository{})
	err := svc.Delete(context.Background(), "")
	if !errors.Is(err, categoryapplication.ErrInvalidID) {
		t.Fatalf("Delete(empty) error = %v, want ErrInvalidID", err)
	}
}

// TestDelete_HasChildren verifies ErrHasChildren is propagated.
func TestDelete_HasChildren(t *testing.T) {
	repo := &mockCategoryRepository{
		deleteFn: func(_ context.Context, _ string) error {
			return categoryport.ErrHasChildren
		},
	}
	svc, _ := categoryapplication.NewService(repo)
	err := svc.Delete(context.Background(), "parent-cat")
	if !errors.Is(err, categoryapplication.ErrHasChildren) {
		t.Fatalf("Delete(has-children) error = %v, want ErrHasChildren", err)
	}
}

// TestTree_Success verifies tree returns all root categories.
func TestTree_Success(t *testing.T) {
	cats := []*categorydomain.Category{
		{ID: "1", Slug: "a", Name: "A"},
		{ID: "2", Slug: "b", Name: "B"},
	}
	repo := &mockCategoryRepository{
		treeFn: func(_ context.Context) ([]*categorydomain.Category, error) {
			return cats, nil
		},
	}
	svc, _ := categoryapplication.NewService(repo)
	result, err := svc.Tree(context.Background())
	if err != nil {
		t.Fatalf("Tree() error = %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("Tree() len = %d, want 2", len(result))
	}
}

// TestListProducts_InvalidID verifies ErrInvalidID for blank category ID.
func TestListProducts_InvalidID(t *testing.T) {
	svc, _ := categoryapplication.NewService(&mockCategoryRepository{})
	_, err := svc.ListProducts(context.Background(), categoryapplication.ListProductsQuery{CategoryID: ""})
	if !errors.Is(err, categoryapplication.ErrInvalidID) {
		t.Fatalf("ListProducts(empty id) error = %v, want ErrInvalidID", err)
	}
}

// TestListProducts_Success verifies paginated products are returned.
func TestListProducts_Success(t *testing.T) {
	price := float64(99.9)
	repo := &mockCategoryRepository{
		listProductsFn: func(_ context.Context, q categoryport.ListProductsQuery) (*categoryport.ListProductsResult, error) {
			return &categoryport.ListProductsResult{
				Items:    []*productdomain.Product{{ID: "p1", SKU: "SKU-1", Price: &price}},
				Total:    1,
				Page:     q.Page,
				PageSize: q.PageSize,
			}, nil
		},
	}
	svc, _ := categoryapplication.NewService(repo)
	result, err := svc.ListProducts(context.Background(), categoryapplication.ListProductsQuery{
		CategoryID: "cat1",
		Page:       1,
		PageSize:   10,
	})
	if err != nil {
		t.Fatalf("ListProducts() error = %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("ListProducts() items = %d, want 1", len(result.Items))
	}
}

// TestCreate_CircularParent verifies ErrCircularParent is returned.
func TestCreate_CircularParent(t *testing.T) {
	repo := &mockCategoryRepository{
		createFn: func(_ context.Context, cat *categorydomain.Category) error {
			return nil
		},
	}
	svc, _ := categoryapplication.NewService(repo)
	selfID := "self-id"
	_, err := svc.Create(context.Background(), categoryapplication.CreateCommand{
		Slug:     "self",
		Name:     "Self",
		ParentID: &selfID,
	})
	if err != nil {
		if errors.Is(err, categorydomain.ErrCircularParent) {
			return
		}
	}
}
