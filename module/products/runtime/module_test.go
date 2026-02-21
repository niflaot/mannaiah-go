package runtime

import (
	"context"
	"errors"
	stdhttp "net/http"
	"testing"

	coredb "mannaiah/module/core/database"
	coredbmigration "mannaiah/module/core/database/migration"
	corehttp "mannaiah/module/core/http"
	productstore "mannaiah/module/products/adapter/store/product"

	"github.com/getkin/kin-openapi/openapi3"
	"gorm.io/gorm"
)

// runtimeAssetLookupMock defines asset lookup behavior for runtime tests.
type runtimeAssetLookupMock struct{}

// Exists returns successful lookup behavior for runtime tests.
func (runtimeAssetLookupMock) Exists(ctx context.Context, id string) (bool, error) {
	return true, nil
}

// TestNewAndRegisterRoutes verifies module wiring and route registration.
func TestNewAndRegisterRoutes(t *testing.T) {
	db := newDBForTest(t)
	module, err := New(db, runtimeAssetLookupMock{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8121}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(module.RegisterRoutes)

	req, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/products", nil)
	resp, testErr := server.App().Test(req)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if resp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, stdhttp.StatusOK)
	}

	variationReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/variations", nil)
	variationResp, variationErr := server.App().Test(variationReq)
	if variationErr != nil {
		t.Fatalf("App().Test(variations) error = %v", variationErr)
	}
	if variationResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("variations status = %d, want %d", variationResp.StatusCode, stdhttp.StatusOK)
	}
}

// TestRegisterRoutesNilModule verifies nil module route registration behavior.
func TestRegisterRoutesNilModule(t *testing.T) {
	var module *Module
	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8122}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}

	server.RegisterRoutes(module.RegisterRoutes)
}

// TestNewRejectsNilDB verifies module constructor validation for nil DB dependencies.
func TestNewRejectsNilDB(t *testing.T) {
	if _, err := New(nil, runtimeAssetLookupMock{}); !errors.Is(err, productstore.ErrNilDB) {
		t.Fatalf("New() error = %v, want ErrNilDB", err)
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

// TestModuleLoad verifies module self-loading behavior for routes and OpenAPI specs.
func TestModuleLoad(t *testing.T) {
	db := newDBForTest(t)
	module, err := New(db, runtimeAssetLookupMock{})
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

// RegisterRoutes ignores registration calls for failing loader probes.
func (failingLoaderProbe) RegisterRoutes(register func(router corehttp.Router)) {}

// AddOpenAPISpec returns a forced merge error.
func (failingLoaderProbe) AddOpenAPISpec(spec *openapi3.T) error {
	return errors.New("merge failed")
}

// TestModuleLoadError verifies loader merge failures are returned.
func TestModuleLoadError(t *testing.T) {
	db := newDBForTest(t)
	module, err := New(db, runtimeAssetLookupMock{})
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
	module, err := New(db, runtimeAssetLookupMock{})
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
	module, err := New(db, runtimeAssetLookupMock{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	module.SetAuthorizer(nil)
}

// TestVariationService verifies variation service exposure behavior.
func TestVariationService(t *testing.T) {
	db := newDBForTest(t)
	module, err := New(db, runtimeAssetLookupMock{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if module.VariationService() == nil {
		t.Fatalf("VariationService() should not be nil")
	}
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
