package assets

import (
	"context"
	"testing"

	"go.uber.org/zap"
	"mannaiah/module/assets/port"
	coredb "mannaiah/module/core/database"
)

// storageMock defines storage behavior for facade tests.
type storageMock struct{}

// Upload ignores upload behavior for facade tests.
func (storageMock) Upload(ctx context.Context, request port.UploadRequest) error { return nil }

// Download ignores download behavior for facade tests.
func (storageMock) Download(ctx context.Context, key string) ([]byte, error) {
	return []byte("payload"), nil
}

// Delete ignores delete behavior for facade tests.
func (storageMock) Delete(ctx context.Context, key string) error { return nil }

// Exists ignores exists behavior for facade tests.
func (storageMock) Exists(ctx context.Context, key string) (bool, error) { return true, nil }

// AvailabilityError ignores availability behavior for facade tests.
func (storageMock) AvailabilityError() error { return nil }

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

	module, err := New(db, storageMock{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if module == nil {
		t.Fatalf("expected module instance")
	}
}

// TestNewWithConfigFacade verifies config-aware facade module-constructor behavior.
func TestNewWithConfigFacade(t *testing.T) {
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

	module, err := NewWithConfig(Config{JPGWorkerEnabled: true}, db, storageMock{}, zap.NewNop())
	if err != nil {
		t.Fatalf("NewWithConfig() error = %v", err)
	}
	if module == nil {
		t.Fatalf("expected module instance")
	}
}
