package runtime

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"mannaiah/module/assets/application"
	"mannaiah/module/assets/port"
	coredb "mannaiah/module/core/database"
	coredbmigration "mannaiah/module/core/database/migration"
	corehttp "mannaiah/module/core/http"

	"github.com/getkin/kin-openapi/openapi3"
	"gorm.io/gorm"
)

// runtimeStorageMock defines storage behavior for runtime tests.
type runtimeStorageMock struct{}

// Upload ignores upload behavior for runtime tests.
func (runtimeStorageMock) Upload(ctx context.Context, request port.UploadRequest) error { return nil }

// Delete ignores delete behavior for runtime tests.
func (runtimeStorageMock) Delete(ctx context.Context, key string) error { return nil }

// Exists ignores exists behavior for runtime tests.
func (runtimeStorageMock) Exists(ctx context.Context, key string) (bool, error) { return true, nil }

// AvailabilityError ignores availability behavior for runtime tests.
func (runtimeStorageMock) AvailabilityError() error { return nil }

// TestNewAndRegisterRoutes verifies module wiring and route registration.
func TestNewAndRegisterRoutes(t *testing.T) {
	db := newDBForTest(t)
	module, err := New(db, runtimeStorageMock{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8131}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(module.RegisterRoutes)

	request, _ := http.NewRequest(http.MethodGet, "/assets", nil)
	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
}

// TestNewRejectsNilDependencies verifies constructor validation for nil dependencies.
func TestNewRejectsNilDependencies(t *testing.T) {
	if _, err := New(nil, runtimeStorageMock{}); err == nil {
		t.Fatalf("expected nil db error")
	}

	db := newDBForTest(t)
	if _, err := New(db, nil); !errors.Is(err, application.ErrNilStorage) {
		t.Fatalf("New(db,nil) error = %v, want application.ErrNilStorage", err)
	}
}

// loaderProbe defines startup loader behavior for module tests.
type loaderProbe struct {
	// registered indicates whether routes were registered.
	registered bool
	// specAdded indicates whether OpenAPI specs were added.
	specAdded bool
}

// RegisterRoutes captures route registration calls.
func (l *loaderProbe) RegisterRoutes(register func(router corehttp.Router)) {
	l.registered = true
}

// AddOpenAPISpec captures OpenAPI spec merge calls.
func (l *loaderProbe) AddOpenAPISpec(spec *openapi3.T) error {
	l.specAdded = spec != nil
	return nil
}

// TestModuleLoad verifies module self-loading behavior.
func TestModuleLoad(t *testing.T) {
	db := newDBForTest(t)
	module, err := New(db, runtimeStorageMock{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	probe := &loaderProbe{}
	if err := module.Load(probe); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !probe.registered {
		t.Fatalf("expected route registration")
	}
	if !probe.specAdded {
		t.Fatalf("expected OpenAPI spec merge")
	}
}

// failingLoaderProbe defines loader behavior that fails while merging specs.
type failingLoaderProbe struct{}

// RegisterRoutes ignores registration calls.
func (failingLoaderProbe) RegisterRoutes(register func(router corehttp.Router)) {}

// AddOpenAPISpec returns a forced merge error.
func (failingLoaderProbe) AddOpenAPISpec(spec *openapi3.T) error {
	return errors.New("merge failed")
}

// TestModuleLoadError verifies loader merge failures are returned.
func TestModuleLoadError(t *testing.T) {
	db := newDBForTest(t)
	module, err := New(db, runtimeStorageMock{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := module.Load(failingLoaderProbe{}); err == nil {
		t.Fatalf("expected load error")
	}
}

// TestModuleLoadNilLoader verifies nil loader behavior.
func TestModuleLoadNilLoader(t *testing.T) {
	db := newDBForTest(t)
	module, err := New(db, runtimeStorageMock{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := module.Load(nil); err != nil {
		t.Fatalf("Load(nil) error = %v", err)
	}
}

// TestSetAuthorizer verifies optional authorizer wiring behavior.
func TestSetAuthorizer(t *testing.T) {
	db := newDBForTest(t)
	module, err := New(db, runtimeStorageMock{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	module.SetAuthorizer(nil)
}

// newDBForTest creates an in-memory DB for module tests.
func newDBForTest(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := coredb.Open(coredb.Config{Driver: "sqlite", DSN: "file::memory:?cache=shared"}, nil)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := coredbmigration.Apply(context.Background(), db, coredbmigration.Config{Enabled: true, Driver: "sqlite", Table: "schema_migrations"}, nil); err != nil {
		t.Fatalf("migration.Apply() error = %v", err)
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
