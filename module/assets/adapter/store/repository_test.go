package store

import (
	"context"
	errorspkg "errors"
	"testing"

	"mannaiah/module/assets/domain"
	"mannaiah/module/assets/port"
	coredb "mannaiah/module/core/database"
)

// TestNewRepository validates repository constructor behavior.
func TestNewRepository(t *testing.T) {
	if _, err := NewRepository(nil); !errorspkg.Is(err, ErrNilDB) {
		t.Fatalf("NewRepository(nil) error = %v, want ErrNilDB", err)
	}
}

// TestRepositoryCRUD verifies repository CRUD and pagination behavior.
func TestRepositoryCRUD(t *testing.T) {
	repository := newRepositoryForTest(t)
	ctx := context.Background()

	asset := &domain.Asset{ID: "a-1", Key: "assets/a-1.png", Name: "Asset One", OriginalName: "one.png", MimeType: "image/png", Size: 120}
	if createErr := repository.Create(ctx, asset); createErr != nil {
		t.Fatalf("Create() error = %v", createErr)
	}

	loaded, getErr := repository.GetByID(ctx, "a-1")
	if getErr != nil {
		t.Fatalf("GetByID() error = %v", getErr)
	}
	if loaded.ID != asset.ID {
		t.Fatalf("loaded.ID = %q, want %q", loaded.ID, asset.ID)
	}

	page, listErr := repository.List(ctx, port.ListQuery{Page: 1, Limit: 10, Filters: "asset"})
	if listErr != nil {
		t.Fatalf("List() error = %v", listErr)
	}
	if page.Total != 1 {
		t.Fatalf("page.Total = %d, want %d", page.Total, 1)
	}

	updated, updateErr := repository.UpdateName(ctx, "a-1", "Asset Renamed")
	if updateErr != nil {
		t.Fatalf("UpdateName() error = %v", updateErr)
	}
	if updated.Name != "Asset Renamed" {
		t.Fatalf("updated.Name = %q, want %q", updated.Name, "Asset Renamed")
	}

	if deleteErr := repository.SoftDelete(ctx, "a-1"); deleteErr != nil {
		t.Fatalf("SoftDelete() error = %v", deleteErr)
	}
	if _, getDeletedErr := repository.GetByID(ctx, "a-1"); !errorspkg.Is(getDeletedErr, port.ErrNotFound) {
		t.Fatalf("GetByID(deleted) error = %v, want port.ErrNotFound", getDeletedErr)
	}
}

// TestRepositoryNotFound verifies not-found behavior across operations.
func TestRepositoryNotFound(t *testing.T) {
	repository := newRepositoryForTest(t)
	ctx := context.Background()

	if _, err := repository.GetByID(ctx, "missing"); !errorspkg.Is(err, port.ErrNotFound) {
		t.Fatalf("GetByID(missing) error = %v, want port.ErrNotFound", err)
	}
	if _, err := repository.UpdateName(ctx, "missing", "name"); !errorspkg.Is(err, port.ErrNotFound) {
		t.Fatalf("UpdateName(missing) error = %v, want port.ErrNotFound", err)
	}
	if err := repository.SoftDelete(ctx, "missing"); !errorspkg.Is(err, port.ErrNotFound) {
		t.Fatalf("SoftDelete(missing) error = %v, want port.ErrNotFound", err)
	}
}

// TestRepositoryDuplicateKey verifies duplicate key create behavior.
func TestRepositoryDuplicateKey(t *testing.T) {
	repository := newRepositoryForTest(t)
	ctx := context.Background()

	if err := repository.Create(ctx, &domain.Asset{ID: "a-1", Key: "assets/a-1.png", Name: "Asset", OriginalName: "one.png", MimeType: "image/png", Size: 120}); err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}
	if err := repository.Create(ctx, &domain.Asset{ID: "a-2", Key: "assets/a-1.png", Name: "Asset", OriginalName: "two.png", MimeType: "image/png", Size: 120}); err == nil {
		t.Fatalf("expected duplicate key error")
	}
}

// TestNormalizePagination verifies pagination helper defaults and limits.
func TestNormalizePagination(t *testing.T) {
	page, limit := normalizePagination(0, 0)
	if page != 1 || limit != 10 {
		t.Fatalf("normalizePagination(0,0) = (%d,%d), want (1,10)", page, limit)
	}
	_, capped := normalizePagination(1, 999)
	if capped != 100 {
		t.Fatalf("normalizePagination limit = %d, want %d", capped, 100)
	}
}

// newRepositoryForTest creates a repository bound to in-memory sqlite.
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
