package product

import (
	"context"
	errorspkg "errors"
	"testing"

	coredb "mannaiah/module/core/database"
	coredbmigration "mannaiah/module/core/database/migration"
	productdomain "mannaiah/module/products/domain/product"
	productport "mannaiah/module/products/port/product"
)

// TestNewRepository validates constructor behavior.
func TestNewRepository(t *testing.T) {
	if _, err := NewRepository(nil); !errorspkg.Is(err, ErrNilDB) {
		t.Fatalf("NewRepository(nil) error = %v, want ErrNilDB", err)
	}
}

// TestRepositoryCRUD verifies create/get/list/update/delete behavior.
func TestRepositoryCRUD(t *testing.T) {
	repository := newRepositoryForTest(t)
	ctx := context.Background()

	seedTagsForTest(t, repository, "bebida", "proteina")

	entity := &productdomain.Product{
		SKU:  "SKU-1",
		Tags: []string{"bebida", "proteina"},
		Gallery: []productdomain.GalleryItem{
			{AssetID: "asset-2", Position: productIntPointer(3), IsMain: false},
			{AssetID: "asset-1", Position: productIntPointer(1), VariationPosition: productIntPointer(0), IsMain: true, IncludedRealms: []string{"b2b"}, VariationIDs: []string{"v1"}},
		},
		Datasheets: []productdomain.Datasheet{
			{Realm: "default", Name: "Name", Description: "Desc", Attributes: map[string]any{"weight": 12}},
		},
		Variations: []string{"v1", "v2"},
		Variants:   []productdomain.Variant{{VariationIDs: []string{"v1"}, SKU: "SKU-1-V1"}},
	}
	entity.Normalize()
	if err := repository.Create(ctx, entity); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if entity.ID == "" {
		t.Fatalf("expected product id after create")
	}

	stored, err := repository.GetByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if stored.SKU != "SKU-1" {
		t.Fatalf("stored.SKU = %q, want %q", stored.SKU, "SKU-1")
	}
	if len(stored.Gallery) != 2 {
		t.Fatalf("stored.Gallery = %#v, want two gallery items", stored.Gallery)
	}
	if stored.Gallery[0].AssetID != "asset-1" || stored.Gallery[1].AssetID != "asset-2" {
		t.Fatalf("stored.Gallery order = %#v, want asset-1 then asset-2 by position", stored.Gallery)
	}
	if stored.Gallery[0].Position == nil || *stored.Gallery[0].Position != 1 {
		t.Fatalf("stored.Gallery[0].Position = %v, want 1", stored.Gallery[0].Position)
	}
	if stored.Gallery[0].VariationPosition == nil || *stored.Gallery[0].VariationPosition != 0 {
		t.Fatalf("stored.Gallery[0].VariationPosition = %v, want 0", stored.Gallery[0].VariationPosition)
	}
	if stored.Gallery[1].Position == nil || *stored.Gallery[1].Position != 3 {
		t.Fatalf("stored.Gallery[1].Position = %v, want 3", stored.Gallery[1].Position)
	}
	if stored.Gallery[1].VariationPosition != nil {
		t.Fatalf("stored.Gallery[1].VariationPosition = %v, want nil", stored.Gallery[1].VariationPosition)
	}
	if len(stored.Datasheets) != 1 || stored.Datasheets[0].Realm != "default" {
		t.Fatalf("stored.Datasheets = %#v, want one default datasheet", stored.Datasheets)
	}
	if len(stored.Variations) != 2 {
		t.Fatalf("stored.Variations = %#v, want two items", stored.Variations)
	}
	if len(stored.Variants) != 1 || stored.Variants[0].SKU != "SKU-1-V1" {
		t.Fatalf("stored.Variants = %#v, want one SKU-1-V1 variant", stored.Variants)
	}
	if len(stored.Tags) != 2 || stored.Tags[0] != "bebida" || stored.Tags[1] != "proteina" {
		t.Fatalf("stored.Tags = %#v, want [bebida proteina]", stored.Tags)
	}

	items, err := repository.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want %d", len(items), 1)
	}

	stored.SKU = "SKU-2"
	if err := repository.Update(ctx, stored); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	updated, err := repository.GetByID(ctx, stored.ID)
	if err != nil {
		t.Fatalf("GetByID(updated) error = %v", err)
	}
	if updated.SKU != "SKU-2" {
		t.Fatalf("updated.SKU = %q, want %q", updated.SKU, "SKU-2")
	}

	if err := repository.Delete(ctx, updated.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := repository.GetByID(ctx, updated.ID); !errorspkg.Is(err, productport.ErrNotFound) {
		t.Fatalf("GetByID(deleted) error = %v, want productport.ErrNotFound", err)
	}
}

// TestRepositoryDuplicateSKU verifies duplicate SKU behavior.
func TestRepositoryDuplicateSKU(t *testing.T) {
	repository := newRepositoryForTest(t)
	ctx := context.Background()

	first := &productdomain.Product{SKU: "SKU-1"}
	first.Normalize()
	if err := repository.Create(ctx, first); err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}

	second := &productdomain.Product{SKU: "SKU-1"}
	second.Normalize()
	if err := repository.Create(ctx, second); !errorspkg.Is(err, productport.ErrDuplicateSKU) {
		t.Fatalf("Create(duplicate) error = %v, want productport.ErrDuplicateSKU", err)
	}
}

// TestRepositoryGetByIDsPreservesInputOrder verifies GetByIDs returns products in requested order.
func TestRepositoryGetByIDsPreservesInputOrder(t *testing.T) {
	repository := newRepositoryForTest(t)
	ctx := context.Background()

	first := &productdomain.Product{SKU: "SKU-ORDER-1"}
	first.Normalize()
	if err := repository.Create(ctx, first); err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}

	second := &productdomain.Product{SKU: "SKU-ORDER-2"}
	second.Normalize()
	if err := repository.Create(ctx, second); err != nil {
		t.Fatalf("Create(second) error = %v", err)
	}

	items, err := repository.GetByIDs(ctx, []string{second.ID, first.ID})
	if err != nil {
		t.Fatalf("GetByIDs() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].ID != second.ID || items[1].ID != first.ID {
		t.Fatalf("GetByIDs() order = [%s %s], want [%s %s]", items[0].ID, items[1].ID, second.ID, first.ID)
	}
}

// TestRepositoryNotFound verifies not-found behavior.
func TestRepositoryNotFound(t *testing.T) {
	repository := newRepositoryForTest(t)
	ctx := context.Background()

	if _, err := repository.GetByID(ctx, "missing"); !errorspkg.Is(err, productport.ErrNotFound) {
		t.Fatalf("GetByID(missing) error = %v, want productport.ErrNotFound", err)
	}
	missing := &productdomain.Product{ID: "missing", SKU: "SKU"}
	if err := repository.Update(ctx, missing); !errorspkg.Is(err, productport.ErrNotFound) {
		t.Fatalf("Update(missing) error = %v, want productport.ErrNotFound", err)
	}
	if err := repository.Delete(ctx, "missing"); !errorspkg.Is(err, productport.ErrNotFound) {
		t.Fatalf("Delete(missing) error = %v, want productport.ErrNotFound", err)
	}
}

// TestMapper verifies mapping helper behavior.
func TestMapper(t *testing.T) {
	raw, err := marshalAttributeValue(map[string]any{"weight": 12})
	if err != nil {
		t.Fatalf("marshalAttributeValue() error = %v", err)
	}
	value, err := unmarshalAttributeValue(raw)
	if err != nil {
		t.Fatalf("unmarshalAttributeValue() error = %v", err)
	}
	if value == nil {
		t.Fatalf("expected non-nil unmarshaled value")
	}
}

// TestGenerateID verifies generated ID behavior.
func TestGenerateID(t *testing.T) {
	if value := generateID(); value == "" {
		t.Fatalf("generateID() should not be empty")
	}
}

// TestIsDuplicateSKUErr verifies duplicate detection helper behavior.
func TestIsDuplicateSKUErr(t *testing.T) {
	if isDuplicateSKUErr(nil) {
		t.Fatalf("expected nil error to be non-duplicate")
	}
	if !isDuplicateSKUErr(errorspkg.New("UNIQUE constraint failed: products.sku")) {
		t.Fatalf("expected sku unique error to be detected")
	}
}

// TestEnsureSchemaNoop verifies repository EnsureSchema does not mutate schema at runtime.
func TestEnsureSchemaNoop(t *testing.T) {
	repository := newRepositoryForTest(t)
	if err := repository.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema() error = %v", err)
	}
}

// newRepositoryForTest creates in-memory repositories for tests.
func newRepositoryForTest(t *testing.T) *Repository {
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

	repository, err := NewRepository(db)
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	if err := coredbmigration.Apply(context.Background(), db, coredbmigration.Config{Enabled: true, Driver: "sqlite", Table: "schema_migrations"}, nil); err != nil {
		t.Fatalf("coredbmigration.Apply() error = %v", err)
	}
	if err := repository.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema() error = %v", err)
	}

	return repository
}

func productIntPointer(value int) *int {
	resolved := value
	return &resolved
}

// seedTagsForTest inserts tags into the canonical registry for tests that bypass the application layer.
func seedTagsForTest(t *testing.T, repository *Repository, tags ...string) {
	t.Helper()
	for _, tag := range tags {
		if err := repository.db.Exec(
			"INSERT INTO tags (name, created_at, updated_at) VALUES (?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",
			tag,
		).Error; err != nil {
			t.Fatalf("seedTagsForTest(%q): %v", tag, err)
		}
	}
}
