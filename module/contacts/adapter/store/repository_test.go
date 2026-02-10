package store

import (
	"context"
	"errors"
	"testing"

	"gorm.io/gorm"
	"mannaiah/module/contacts/domain"
	"mannaiah/module/contacts/port"
	coredb "mannaiah/module/core/database"
)

// TestNewRepositoryRejectsNilDB verifies constructor validation for nil DB dependencies.
func TestNewRepositoryRejectsNilDB(t *testing.T) {
	_, err := NewRepository(nil)
	if !errors.Is(err, ErrNilDB) {
		t.Fatalf("NewRepository() error = %v, want ErrNilDB", err)
	}
}

// TestRepositoryCRUDLifecycle verifies create/get/update/delete repository behavior.
func TestRepositoryCRUDLifecycle(t *testing.T) {
	repository := newRepositoryForTest(t)

	entity := &domain.Contact{Email: "john@example.com", FirstName: "John", LastName: "Doe"}
	if err := repository.Create(context.Background(), entity); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if entity.ID == "" {
		t.Fatalf("expected generated id")
	}

	stored, err := repository.GetByID(context.Background(), entity.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if stored.Email != "john@example.com" {
		t.Fatalf("Email = %q, want %q", stored.Email, "john@example.com")
	}

	stored.LegalName = "Acme SAS"
	stored.FirstName = ""
	stored.LastName = ""
	if err := repository.Update(context.Background(), stored); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	updated, err := repository.GetByID(context.Background(), entity.ID)
	if err != nil {
		t.Fatalf("GetByID() after update error = %v", err)
	}
	if updated.LegalName != "Acme SAS" {
		t.Fatalf("LegalName = %q, want %q", updated.LegalName, "Acme SAS")
	}

	if err := repository.Delete(context.Background(), entity.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := repository.GetByID(context.Background(), entity.ID); !errors.Is(err, port.ErrNotFound) {
		t.Fatalf("GetByID() after delete error = %v, want ErrNotFound", err)
	}
}

// TestRepositoryListPaginationAndExclusions verifies list pagination counts filtered totals after exclusions.
func TestRepositoryListPaginationAndExclusions(t *testing.T) {
	repository := newRepositoryForTest(t)

	seed := []domain.Contact{
		{Email: "a@example.com", FirstName: "A", LastName: "A"},
		{Email: "b@example.com", FirstName: "B", LastName: "B"},
		{Email: "c@example.com", FirstName: "C", LastName: "C"},
		{Email: "d@example.com", FirstName: "D", LastName: "D"},
	}
	for index := range seed {
		if err := repository.Create(context.Background(), &seed[index]); err != nil {
			t.Fatalf("Create() seed error = %v", err)
		}
	}

	rows, total, err := repository.List(context.Background(), port.ListQuery{
		Page:       1,
		Limit:      2,
		OrderBy:    "email",
		OrderDir:   "asc",
		ExcludeIDs: []string{seed[1].ID},
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if total != 3 {
		t.Fatalf("total = %d, want %d", total, 3)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want %d", len(rows), 2)
	}
	if rows[0].Email != "a@example.com" || rows[1].Email != "c@example.com" {
		t.Fatalf("unexpected rows order: [%s, %s]", rows[0].Email, rows[1].Email)
	}
}

// TestRepositoryListFiltersByEmail verifies email filtering behavior.
func TestRepositoryListFiltersByEmail(t *testing.T) {
	repository := newRepositoryForTest(t)

	if err := repository.Create(context.Background(), &domain.Contact{Email: "x@example.com", FirstName: "X", LastName: "X"}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := repository.Create(context.Background(), &domain.Contact{Email: "y@example.com", FirstName: "Y", LastName: "Y"}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	rows, total, err := repository.List(context.Background(), port.ListQuery{Email: "y@example.com"})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 1 || len(rows) != 1 {
		t.Fatalf("unexpected filtered result: total=%d len=%d", total, len(rows))
	}
	if rows[0].Email != "y@example.com" {
		t.Fatalf("Email = %q, want %q", rows[0].Email, "y@example.com")
	}
}

// TestRepositoryUpdateDeleteNotFound verifies missing-row update/delete behavior.
func TestRepositoryUpdateDeleteNotFound(t *testing.T) {
	repository := newRepositoryForTest(t)

	updateErr := repository.Update(context.Background(), &domain.Contact{ID: "missing", Email: "x@example.com", LegalName: "Acme"})
	if !errors.Is(updateErr, port.ErrNotFound) {
		t.Fatalf("Update() error = %v, want ErrNotFound", updateErr)
	}

	deleteErr := repository.Delete(context.Background(), "missing")
	if !errors.Is(deleteErr, port.ErrNotFound) {
		t.Fatalf("Delete() error = %v, want ErrNotFound", deleteErr)
	}
}

// TestRepositoryErrorPathsOnClosedDB verifies wrapped DB errors after sql handle closure.
func TestRepositoryErrorPathsOnClosedDB(t *testing.T) {
	db := newDBForTest(t)
	repository, err := NewRepository(db)
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	if err := repository.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema() error = %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("DB() error = %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if err := repository.EnsureSchema(context.Background()); err == nil {
		t.Fatalf("expected EnsureSchema() error on closed db")
	}
	if err := repository.Create(context.Background(), &domain.Contact{Email: "z@example.com", FirstName: "Z", LastName: "Z"}); err == nil {
		t.Fatalf("expected Create() error on closed db")
	}
	if _, err := repository.GetByID(context.Background(), "missing"); err == nil {
		t.Fatalf("expected GetByID() error on closed db")
	}
	if _, _, err := repository.List(context.Background(), port.ListQuery{}); err == nil {
		t.Fatalf("expected List() error on closed db")
	}
	if err := repository.Update(context.Background(), &domain.Contact{ID: "c-1", Email: "a@example.com", LegalName: "Acme"}); err == nil {
		t.Fatalf("expected Update() error on closed db")
	}
	if err := repository.Delete(context.Background(), "c-1"); err == nil {
		t.Fatalf("expected Delete() error on closed db")
	}
}

// newRepositoryForTest creates a schema-ready repository for tests.
func newRepositoryForTest(t *testing.T) *Repository {
	t.Helper()

	db := newDBForTest(t)
	repository, err := NewRepository(db)
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	if err := repository.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema() error = %v", err)
	}

	return repository
}

// newDBForTest creates an in-memory sqlite DB for repository tests.
func newDBForTest(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := coredb.Open(coredb.Config{Driver: "sqlite", DSN: "file::memory:?cache=shared"}, nil)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("DB() error = %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	return db
}
