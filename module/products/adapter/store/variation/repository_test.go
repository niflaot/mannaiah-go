package variation

import (
	"context"
	errorspkg "errors"
	"testing"
	"time"

	coredb "mannaiah/module/core/database"
	variationdomain "mannaiah/module/products/domain/variation"
	variationport "mannaiah/module/products/port/variation"
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

	entity := &variationdomain.Variation{Name: "Red", Definition: variationdomain.DefinitionColor, Value: "#FF0000"}
	entity.Normalize()
	if err := repository.Create(ctx, entity); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if entity.ID == "" {
		t.Fatalf("expected variation id after create")
	}

	stored, err := repository.GetByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if stored.Name != "Red" {
		t.Fatalf("stored.Name = %q, want %q", stored.Name, "Red")
	}

	items, err := repository.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want %d", len(items), 1)
	}

	stored.Name = "Dark Red"
	stored.Value = "#8B0000"
	if err := repository.Update(ctx, stored); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	updated, err := repository.GetByID(ctx, stored.ID)
	if err != nil {
		t.Fatalf("GetByID(updated) error = %v", err)
	}
	if updated.Name != "Dark Red" {
		t.Fatalf("updated.Name = %q, want %q", updated.Name, "Dark Red")
	}
	if updated.Definition != variationdomain.DefinitionColor {
		t.Fatalf("updated.Definition = %q, want immutable %q", updated.Definition, variationdomain.DefinitionColor)
	}

	if err := repository.Delete(ctx, updated.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := repository.GetByID(ctx, updated.ID); !errorspkg.Is(err, variationport.ErrNotFound) {
		t.Fatalf("GetByID(deleted) error = %v, want variationport.ErrNotFound", err)
	}
}

// TestRepositoryNotFound verifies not-found behavior.
func TestRepositoryNotFound(t *testing.T) {
	repository := newRepositoryForTest(t)
	ctx := context.Background()

	if _, err := repository.GetByID(ctx, "missing"); !errorspkg.Is(err, variationport.ErrNotFound) {
		t.Fatalf("GetByID(missing) error = %v, want variationport.ErrNotFound", err)
	}
	missing := &variationdomain.Variation{ID: "missing", Name: "Red", Definition: variationdomain.DefinitionColor, Value: "#FF0000"}
	if err := repository.Update(ctx, missing); !errorspkg.Is(err, variationport.ErrNotFound) {
		t.Fatalf("Update(missing) error = %v, want variationport.ErrNotFound", err)
	}
	if err := repository.Delete(ctx, "missing"); !errorspkg.Is(err, variationport.ErrNotFound) {
		t.Fatalf("Delete(missing) error = %v, want variationport.ErrNotFound", err)
	}
}

// TestMappingHelpers verifies mapping helper behavior.
func TestMappingHelpers(t *testing.T) {
	now := time.Now()
	entity := variationdomain.Variation{ID: "v-1", Name: "Red", Definition: variationdomain.DefinitionColor, Value: "#FF0000", DeletedAt: &now}
	record := toRecord(entity)
	if !record.DeletedAt.Valid {
		t.Fatalf("record.DeletedAt.Valid = false, want true")
	}

	mapped := toDomain(record)
	if mapped.ID != "v-1" {
		t.Fatalf("mapped.ID = %q, want %q", mapped.ID, "v-1")
	}
	if mapped.DeletedAt == nil {
		t.Fatalf("expected mapped deletedAt")
	}
}

// TestGenerateID verifies generated id behavior.
func TestGenerateID(t *testing.T) {
	if value := generateID(); value == "" {
		t.Fatalf("generateID() should not be empty")
	}
}

// newRepositoryForTest creates an in-memory repository for tests.
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
