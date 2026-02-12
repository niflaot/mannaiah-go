package products

import (
	"context"
	"testing"

	coredb "mannaiah/module/core/database"
)

// lookupMock defines asset lookup behavior for facade tests.
type lookupMock struct{}

// Exists returns successful lookup behavior for facade tests.
func (lookupMock) Exists(ctx context.Context, id string) (bool, error) {
	return true, nil
}

// TestOpenAPISpecFacade verifies root facade OpenAPI delegation behavior.
func TestOpenAPISpecFacade(t *testing.T) {
	spec := OpenAPISpec()
	if spec == nil {
		t.Fatalf("OpenAPISpec() should not return nil")
	}
	if spec.OpenAPI != "3.0.3" {
		t.Fatalf("spec.OpenAPI = %q, want %q", spec.OpenAPI, "3.0.3")
	}
}

// TestNewFacade verifies root facade module-constructor behavior.
func TestNewFacade(t *testing.T) {
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

	module, err := New(db, lookupMock{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if module == nil {
		t.Fatalf("expected module instance")
	}
}
