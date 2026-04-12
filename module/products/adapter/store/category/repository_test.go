package category_test

import (
	"context"
	"errors"
	"testing"

	coredb "mannaiah/module/core/database"
	coredbmigration "mannaiah/module/core/database/migration"
	categorystore "mannaiah/module/products/adapter/store/category"
	productstore "mannaiah/module/products/adapter/store/product"
	tagstore "mannaiah/module/products/adapter/store/tag"
	categorydomain "mannaiah/module/products/domain/category"
	productdomain "mannaiah/module/products/domain/product"
	categoryport "mannaiah/module/products/port/category"
)

// newCategoryRepositoryForTest creates in-memory category repositories for tests.
func newCategoryRepositoryForTest(t *testing.T) *categorystore.Repository {
	t.Helper()

	db, err := coredb.Open(coredb.Config{Driver: "sqlite", DSN: "file::memory:?cache=shared"}, nil)
	if err != nil {
		t.Fatalf("coredb.Open() error = %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	if err := coredbmigration.Apply(context.Background(), db, coredbmigration.Config{Enabled: true, Driver: "sqlite", Table: "schema_migrations"}, nil); err != nil {
		t.Fatalf("coredbmigration.Apply() error = %v", err)
	}

	repo, err := categorystore.NewRepository(db)
	if err != nil {
		t.Fatalf("categorystore.NewRepository() error = %v", err)
	}

	return repo
}

// createTestProduct creates a product for use in category tests.
func createTestProduct(t *testing.T, db interface {
	DB() interface{ Close() error }
}, sku string, price *float64, tags []string) string {
	t.Helper()

	return ""
}

// TestNewCategoryRepository_NilDB verifies ErrNilDB is returned.
func TestNewCategoryRepository_NilDB(t *testing.T) {
	_, err := categorystore.NewRepository(nil)
	if !errors.Is(err, categorystore.ErrNilDB) {
		t.Fatalf("NewRepository(nil) error = %v, want ErrNilDB", err)
	}
}

// TestCategoryRepositoryCRUD verifies full category lifecycle.
func TestCategoryRepositoryCRUD(t *testing.T) {
	repo := newCategoryRepositoryForTest(t)
	ctx := context.Background()

	cat := &categorydomain.Category{
		ID:          "cat-1",
		Slug:        "electronics",
		Name:        "Electronics",
		Description: "Electronic goods",
		Filter: categorydomain.Filter{
			Tags: []string{"tech", "gadget"},
		},
	}

	if err := repo.Create(ctx, cat); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	stored, err := repo.GetByID(ctx, "cat-1")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if stored.Slug != "electronics" {
		t.Fatalf("Slug = %q, want %q", stored.Slug, "electronics")
	}
	if len(stored.Filter.Tags) != 2 {
		t.Fatalf("Filter.Tags = %v, want 2 tags", stored.Filter.Tags)
	}

	bySlug, err := repo.GetBySlug(ctx, "electronics")
	if err != nil {
		t.Fatalf("GetBySlug() error = %v", err)
	}
	if bySlug.ID != "cat-1" {
		t.Fatalf("GetBySlug().ID = %q, want %q", bySlug.ID, "cat-1")
	}

	stored.Name = "Electronics Updated"
	if err := repo.Update(ctx, stored); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	updated, err := repo.GetByID(ctx, "cat-1")
	if err != nil {
		t.Fatalf("GetByID(updated) error = %v", err)
	}
	if updated.Name != "Electronics Updated" {
		t.Fatalf("Name = %q, want %q", updated.Name, "Electronics Updated")
	}

	if err := repo.Delete(ctx, "cat-1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := repo.GetByID(ctx, "cat-1"); !errors.Is(err, categoryport.ErrNotFound) {
		t.Fatalf("GetByID(deleted) error = %v, want ErrNotFound", err)
	}
}

// TestCategoryRepository_DuplicateSlug verifies duplicate slug behavior.
func TestCategoryRepository_DuplicateSlug(t *testing.T) {
	repo := newCategoryRepositoryForTest(t)
	ctx := context.Background()

	a := &categorydomain.Category{ID: "a", Slug: "dup-slug", Name: "A"}
	if err := repo.Create(ctx, a); err != nil {
		t.Fatalf("Create(a) error = %v", err)
	}

	b := &categorydomain.Category{ID: "b", Slug: "dup-slug", Name: "B"}
	if err := repo.Create(ctx, b); !errors.Is(err, categoryport.ErrDuplicateSlug) {
		t.Fatalf("Create(dup) error = %v, want ErrDuplicateSlug", err)
	}
}

// TestCategoryRepository_NotFound verifies not-found behavior.
func TestCategoryRepository_NotFound(t *testing.T) {
	repo := newCategoryRepositoryForTest(t)
	ctx := context.Background()

	if _, err := repo.GetByID(ctx, "missing"); !errors.Is(err, categoryport.ErrNotFound) {
		t.Fatalf("GetByID(missing) error = %v, want ErrNotFound", err)
	}
	if _, err := repo.GetBySlug(ctx, "missing-slug"); !errors.Is(err, categoryport.ErrNotFound) {
		t.Fatalf("GetBySlug(missing) error = %v, want ErrNotFound", err)
	}
}

// TestCategoryRepository_Tree verifies tree returns root categories only.
func TestCategoryRepository_Tree(t *testing.T) {
	repo := newCategoryRepositoryForTest(t)
	ctx := context.Background()

	root := &categorydomain.Category{ID: "root", Slug: "root", Name: "Root"}
	if err := repo.Create(ctx, root); err != nil {
		t.Fatalf("Create(root) error = %v", err)
	}
	parentID := "root"
	child := &categorydomain.Category{ID: "child", Slug: "child", Name: "Child", ParentID: &parentID}
	if err := repo.Create(ctx, child); err != nil {
		t.Fatalf("Create(child) error = %v", err)
	}

	tree, err := repo.Tree(ctx)
	if err != nil {
		t.Fatalf("Tree() error = %v", err)
	}
	if len(tree) != 1 || tree[0].ID != "root" {
		t.Fatalf("Tree() = %v, want one root", tree)
	}
}

// TestCategoryRepository_ListChildren verifies children listing.
func TestCategoryRepository_ListChildren(t *testing.T) {
	repo := newCategoryRepositoryForTest(t)
	ctx := context.Background()

	root := &categorydomain.Category{ID: "r", Slug: "r", Name: "Root"}
	if err := repo.Create(ctx, root); err != nil {
		t.Fatalf("Create(root) error = %v", err)
	}
	parentID := "r"
	child1 := &categorydomain.Category{ID: "c1", Slug: "c1", Name: "Child1", ParentID: &parentID}
	child2 := &categorydomain.Category{ID: "c2", Slug: "c2", Name: "Child2", ParentID: &parentID}
	if err := repo.Create(ctx, child1); err != nil {
		t.Fatalf("Create(child1) error = %v", err)
	}
	if err := repo.Create(ctx, child2); err != nil {
		t.Fatalf("Create(child2) error = %v", err)
	}

	children, err := repo.ListChildren(ctx, "r")
	if err != nil {
		t.Fatalf("ListChildren() error = %v", err)
	}
	if len(children) != 2 {
		t.Fatalf("ListChildren() len = %d, want 2", len(children))
	}
}

// TestCategoryRepository_HasChildren verifies ErrHasChildren on delete.
func TestCategoryRepository_HasChildren(t *testing.T) {
	repo := newCategoryRepositoryForTest(t)
	ctx := context.Background()

	root := &categorydomain.Category{ID: "rp", Slug: "rp", Name: "Root Parent"}
	if err := repo.Create(ctx, root); err != nil {
		t.Fatalf("Create(root) error = %v", err)
	}
	parentID := "rp"
	child := &categorydomain.Category{ID: "cp", Slug: "cp", Name: "Child", ParentID: &parentID}
	if err := repo.Create(ctx, child); err != nil {
		t.Fatalf("Create(child) error = %v", err)
	}

	if err := repo.Delete(ctx, "rp"); !errors.Is(err, categoryport.ErrHasChildren) {
		t.Fatalf("Delete(with-children) error = %v, want ErrHasChildren", err)
	}
}

// TestCategoryRepository_PriceRange verifies price range filter persistence.
func TestCategoryRepository_PriceRange(t *testing.T) {
	repo := newCategoryRepositoryForTest(t)
	ctx := context.Background()

	min := float64(10)
	max := float64(200)
	cat := &categorydomain.Category{
		ID:   "pr-cat",
		Slug: "priced",
		Name: "Priced",
		Filter: categorydomain.Filter{
			PriceRange: &categorydomain.PriceRange{Min: &min, Max: &max},
		},
	}
	if err := repo.Create(ctx, cat); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	loaded, err := repo.GetByID(ctx, "pr-cat")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if loaded.Filter.PriceRange == nil {
		t.Fatal("PriceRange is nil after load")
	}
	if *loaded.Filter.PriceRange.Min != min {
		t.Fatalf("PriceRange.Min = %v, want %v", *loaded.Filter.PriceRange.Min, min)
	}
	if *loaded.Filter.PriceRange.Max != max {
		t.Fatalf("PriceRange.Max = %v, want %v", *loaded.Filter.PriceRange.Max, max)
	}
}

// TestCategoryRepository_ListProducts_Empty verifies empty result when no filters/pins.
func TestCategoryRepository_ListProducts_Empty(t *testing.T) {
	repo := newCategoryRepositoryForTest(t)
	ctx := context.Background()

	cat := &categorydomain.Category{ID: "empty-cat", Slug: "empty", Name: "Empty"}
	if err := repo.Create(ctx, cat); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	result, err := repo.ListProducts(ctx, categoryport.ListProductsQuery{CategoryID: "empty-cat", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListProducts() error = %v", err)
	}
	if result.Total != 0 {
		t.Fatalf("Total = %d, want 0", result.Total)
	}
}

// TestCategoryRepository_ListProducts_Pinned verifies pinned products are returned.
func TestCategoryRepository_ListProducts_Pinned(t *testing.T) {
	db, err := coredb.Open(coredb.Config{Driver: "sqlite", DSN: "file::memory:?cache=shared"}, nil)
	if err != nil {
		t.Fatalf("coredb.Open() error = %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := coredbmigration.Apply(context.Background(), db, coredbmigration.Config{Enabled: true, Driver: "sqlite", Table: "schema_migrations"}, nil); err != nil {
		t.Fatalf("coredbmigration.Apply() error = %v", err)
	}

	catRepo, err := categorystore.NewRepository(db)
	if err != nil {
		t.Fatalf("categorystore.NewRepository() error = %v", err)
	}
	prodRepo, err := productstore.NewRepository(db)
	if err != nil {
		t.Fatalf("productstore.NewRepository() error = %v", err)
	}

	ctx := context.Background()
	firstProduct := &productdomain.Product{SKU: "PINNED-1"}
	firstProduct.Normalize()
	if err := prodRepo.Create(ctx, firstProduct); err != nil {
		t.Fatalf("product.Create(firstProduct) error = %v", err)
	}
	secondProduct := &productdomain.Product{SKU: "PINNED-2"}
	secondProduct.Normalize()
	if err := prodRepo.Create(ctx, secondProduct); err != nil {
		t.Fatalf("product.Create(secondProduct) error = %v", err)
	}

	cat := &categorydomain.Category{
		ID:         "pinned-cat",
		Slug:       "pinned",
		Name:       "Pinned",
		ProductIDs: []string{secondProduct.ID, firstProduct.ID},
	}
	if err := catRepo.Create(ctx, cat); err != nil {
		t.Fatalf("Create(category) error = %v", err)
	}

	result, err := catRepo.ListProducts(ctx, categoryport.ListProductsQuery{CategoryID: "pinned-cat", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListProducts() error = %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("Total = %d, want 2", result.Total)
	}
	if len(result.Items) != 2 {
		t.Fatalf("Items len = %d, want 2", len(result.Items))
	}
	if result.Items[0].ID != secondProduct.ID || result.Items[1].ID != firstProduct.ID {
		t.Fatalf("Items order = [%s %s], want [%s %s]", result.Items[0].ID, result.Items[1].ID, secondProduct.ID, firstProduct.ID)
	}
}

// TestCategoryRepository_ListProducts_IncludeChildrenPinned verifies parent categories include child pinned products when includeChildren is enabled.
func TestCategoryRepository_ListProducts_IncludeChildrenPinned(t *testing.T) {
	db, err := coredb.Open(coredb.Config{Driver: "sqlite", DSN: "file::memory:?cache=shared"}, nil)
	if err != nil {
		t.Fatalf("coredb.Open() error = %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := coredbmigration.Apply(context.Background(), db, coredbmigration.Config{Enabled: true, Driver: "sqlite", Table: "schema_migrations"}, nil); err != nil {
		t.Fatalf("coredbmigration.Apply() error = %v", err)
	}

	catRepo, err := categorystore.NewRepository(db)
	if err != nil {
		t.Fatalf("categorystore.NewRepository() error = %v", err)
	}
	prodRepo, err := productstore.NewRepository(db)
	if err != nil {
		t.Fatalf("productstore.NewRepository() error = %v", err)
	}

	ctx := context.Background()
	product := &productdomain.Product{SKU: "CHILD-PINNED-1"}
	if err := prodRepo.Create(ctx, product); err != nil {
		t.Fatalf("product.Create() error = %v", err)
	}

	parent := &categorydomain.Category{
		ID:              "parent-cat",
		Slug:            "parent-cat",
		Name:            "Parent Cat",
		IncludeChildren: true,
	}
	if err := catRepo.Create(ctx, parent); err != nil {
		t.Fatalf("Create(parent) error = %v", err)
	}
	parentID := parent.ID
	child := &categorydomain.Category{
		ID:         "child-cat",
		Slug:       "child-cat",
		Name:       "Child Cat",
		ParentID:   &parentID,
		ProductIDs: []string{product.ID},
	}
	if err := catRepo.Create(ctx, child); err != nil {
		t.Fatalf("Create(child) error = %v", err)
	}

	result, err := catRepo.ListProducts(ctx, categoryport.ListProductsQuery{CategoryID: parent.ID, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListProducts() error = %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("Total = %d, want 1", result.Total)
	}
	if len(result.Items) != 1 || result.Items[0].ID != product.ID {
		t.Fatalf("Items = %#v, want child pinned product %q", result.Items, product.ID)
	}
}

// TestCategoryRepository_ListProducts_IncludeChildrenPinnedPreservesCategoryOrder verifies child category product order is preserved.
func TestCategoryRepository_ListProducts_IncludeChildrenPinnedPreservesCategoryOrder(t *testing.T) {
	db, err := coredb.Open(coredb.Config{Driver: "sqlite", DSN: "file::memory:?cache=shared"}, nil)
	if err != nil {
		t.Fatalf("coredb.Open() error = %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := coredbmigration.Apply(context.Background(), db, coredbmigration.Config{Enabled: true, Driver: "sqlite", Table: "schema_migrations"}, nil); err != nil {
		t.Fatalf("coredbmigration.Apply() error = %v", err)
	}

	catRepo, err := categorystore.NewRepository(db)
	if err != nil {
		t.Fatalf("categorystore.NewRepository() error = %v", err)
	}
	prodRepo, err := productstore.NewRepository(db)
	if err != nil {
		t.Fatalf("productstore.NewRepository() error = %v", err)
	}

	ctx := context.Background()
	urbanFirst := &productdomain.Product{SKU: "URBAN-1"}
	urbanFirst.Normalize()
	if err := prodRepo.Create(ctx, urbanFirst); err != nil {
		t.Fatalf("product.Create(urbanFirst) error = %v", err)
	}
	urbanSecond := &productdomain.Product{SKU: "URBAN-2"}
	urbanSecond.Normalize()
	if err := prodRepo.Create(ctx, urbanSecond); err != nil {
		t.Fatalf("product.Create(urbanSecond) error = %v", err)
	}
	totepack := &productdomain.Product{SKU: "TOTE-1"}
	totepack.Normalize()
	if err := prodRepo.Create(ctx, totepack); err != nil {
		t.Fatalf("product.Create(totepack) error = %v", err)
	}

	parent := &categorydomain.Category{
		ID:              "parent-order-cat",
		Slug:            "parent-order-cat",
		Name:            "Parent Order Cat",
		IncludeChildren: true,
	}
	if err := catRepo.Create(ctx, parent); err != nil {
		t.Fatalf("Create(parent) error = %v", err)
	}
	parentID := parent.ID
	urban := &categorydomain.Category{
		ID:         "urban-order-cat",
		Slug:       "urban-order-cat",
		Name:       "Urban Order Cat",
		ParentID:   &parentID,
		ProductIDs: []string{urbanFirst.ID, urbanSecond.ID},
	}
	if err := catRepo.Create(ctx, urban); err != nil {
		t.Fatalf("Create(urban) error = %v", err)
	}
	tote := &categorydomain.Category{
		ID:         "tote-order-cat",
		Slug:       "tote-order-cat",
		Name:       "Tote Order Cat",
		ParentID:   &parentID,
		ProductIDs: []string{totepack.ID},
	}
	if err := catRepo.Create(ctx, tote); err != nil {
		t.Fatalf("Create(tote) error = %v", err)
	}

	result, err := catRepo.ListProducts(ctx, categoryport.ListProductsQuery{CategoryID: parent.ID, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListProducts() error = %v", err)
	}
	if len(result.Items) != 3 {
		t.Fatalf("Items len = %d, want 3", len(result.Items))
	}
	if result.Items[0].ID != urbanFirst.ID || result.Items[1].ID != urbanSecond.ID || result.Items[2].ID != totepack.ID {
		t.Fatalf("Items order = [%s %s %s], want [%s %s %s]", result.Items[0].ID, result.Items[1].ID, result.Items[2].ID, urbanFirst.ID, urbanSecond.ID, totepack.ID)
	}
}

// TestCategoryRepository_ListProducts_IncludeChildrenFilters verifies parent categories include child filter-resolved products when includeChildren is enabled.
func TestCategoryRepository_ListProducts_IncludeChildrenFilters(t *testing.T) {
	db, err := coredb.Open(coredb.Config{Driver: "sqlite", DSN: "file::memory:?cache=shared"}, nil)
	if err != nil {
		t.Fatalf("coredb.Open() error = %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := coredbmigration.Apply(context.Background(), db, coredbmigration.Config{Enabled: true, Driver: "sqlite", Table: "schema_migrations"}, nil); err != nil {
		t.Fatalf("coredbmigration.Apply() error = %v", err)
	}

	catRepo, err := categorystore.NewRepository(db)
	if err != nil {
		t.Fatalf("categorystore.NewRepository() error = %v", err)
	}
	prodRepo, err := productstore.NewRepository(db)
	if err != nil {
		t.Fatalf("productstore.NewRepository() error = %v", err)
	}
	tagsRepo, err := tagstore.NewRepository(db)
	if err != nil {
		t.Fatalf("tagstore.NewRepository() error = %v", err)
	}

	ctx := context.Background()
	if err := tagsRepo.EnsureAll(ctx, []string{"morrales"}); err != nil {
		t.Fatalf("tags.EnsureAll() error = %v", err)
	}
	product := &productdomain.Product{
		SKU:  "CHILD-FILTER-1",
		Tags: []string{"morrales"},
	}
	if err := prodRepo.Create(ctx, product); err != nil {
		t.Fatalf("product.Create() error = %v", err)
	}

	parent := &categorydomain.Category{
		ID:              "parent-filter-cat",
		Slug:            "parent-filter-cat",
		Name:            "Parent Filter Cat",
		IncludeChildren: true,
	}
	if err := catRepo.Create(ctx, parent); err != nil {
		t.Fatalf("Create(parent) error = %v", err)
	}
	parentID := parent.ID
	child := &categorydomain.Category{
		ID:       "child-filter-cat",
		Slug:     "child-filter-cat",
		Name:     "Child Filter Cat",
		ParentID: &parentID,
		Filter: categorydomain.Filter{
			Tags: []string{"morrales"},
		},
	}
	if err := catRepo.Create(ctx, child); err != nil {
		t.Fatalf("Create(child) error = %v", err)
	}

	result, err := catRepo.ListProducts(ctx, categoryport.ListProductsQuery{CategoryID: parent.ID, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListProducts() error = %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("Total = %d, want 1", result.Total)
	}
	if len(result.Items) != 1 || result.Items[0].ID != product.ID {
		t.Fatalf("Items = %#v, want child filter product %q", result.Items, product.ID)
	}
}

// TestCategoryRepository_EnsureSchemaNoop verifies EnsureSchema does nothing.
func TestCategoryRepository_EnsureSchemaNoop(t *testing.T) {
	repo := newCategoryRepositoryForTest(t)
	if err := repo.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema() error = %v", err)
	}
}
