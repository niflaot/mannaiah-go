package product

import (
	"context"
	errorspkg "errors"
	"testing"

	coredb "mannaiah/module/core/database"
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

	entity := &productdomain.Product{
		SKU: "SKU-1",
		Gallery: []productdomain.GalleryItem{
			{AssetID: "asset-1", IsMain: true, ExcludedRealms: []string{"b2b"}, VariationIDs: []string{"v1"}},
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
	if len(stored.Gallery) != 1 || stored.Gallery[0].AssetID != "asset-1" {
		t.Fatalf("stored.Gallery = %#v, want one asset-1 item", stored.Gallery)
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

// TestEnsureSchemaMigratesLegacyJSONRelations verifies legacy JSON migration into normalized tables.
func TestEnsureSchemaMigratesLegacyJSONRelations(t *testing.T) {
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

	ctx := context.Background()
	if err := db.WithContext(ctx).Exec(`
		CREATE TABLE products (
			id TEXT PRIMARY KEY,
			sku TEXT NOT NULL UNIQUE,
			gallery TEXT,
			datasheets TEXT,
			variations TEXT,
			variants TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);
	`).Error; err != nil {
		t.Fatalf("create legacy products table error = %v", err)
	}
	if err := db.WithContext(ctx).Exec(`
		INSERT INTO products (id, sku, gallery, datasheets, variations, variants, created_at, updated_at)
		VALUES (
			'p-legacy',
			'SKU-LEGACY',
			'[{"assetId":"asset-1","isMain":true,"excludedRealms":["b2b"],"variationIds":["v1"]}]',
			'[{"realm":"default","name":"Legacy","description":"desc","attributes":{"weight":10}}]',
			'["v1","v2"]',
			'[{"variationIds":["v1"],"sku":"SKU-LEGACY-V1"}]',
			CURRENT_TIMESTAMP,
			CURRENT_TIMESTAMP
		);
	`).Error; err != nil {
		t.Fatalf("insert legacy product row error = %v", err)
	}

	repository, err := NewRepository(db)
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	if err := repository.EnsureSchema(ctx); err != nil {
		t.Fatalf("EnsureSchema() error = %v", err)
	}

	entity, err := repository.GetByID(ctx, "p-legacy")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if len(entity.Gallery) != 1 || entity.Gallery[0].AssetID != "asset-1" {
		t.Fatalf("entity.Gallery = %#v, want one asset-1 item", entity.Gallery)
	}
	if len(entity.Datasheets) != 1 || entity.Datasheets[0].Realm != "default" {
		t.Fatalf("entity.Datasheets = %#v, want one default datasheet", entity.Datasheets)
	}
	if len(entity.Variations) != 2 {
		t.Fatalf("entity.Variations = %#v, want two items", entity.Variations)
	}
	if len(entity.Variants) != 1 || entity.Variants[0].SKU != "SKU-LEGACY-V1" {
		t.Fatalf("entity.Variants = %#v, want one SKU-LEGACY-V1 variant", entity.Variants)
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
	if err := repository.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema() error = %v", err)
	}

	return repository
}
