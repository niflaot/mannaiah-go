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

	entity := &productdomain.Product{SKU: "SKU-1", Gallery: []productdomain.GalleryItem{{AssetID: "asset-1"}}}
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
	entity := productdomain.Product{SKU: "SKU", Gallery: []productdomain.GalleryItem{{AssetID: "asset"}}}
	payload, err := toRecordPayload(entity)
	if err != nil {
		t.Fatalf("toRecordPayload() error = %v", err)
	}
	if payload.Gallery == "" {
		t.Fatalf("expected encoded gallery payload")
	}

	mapped, mapErr := toDomain(productRecord{ID: "p-1", SKU: "SKU", Gallery: payload.Gallery, Datasheets: payload.Datasheets, Variations: payload.Variations, Variants: payload.Variants})
	if mapErr != nil {
		t.Fatalf("toDomain() error = %v", mapErr)
	}
	if mapped.ID != "p-1" {
		t.Fatalf("mapped.ID = %q, want %q", mapped.ID, "p-1")
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
