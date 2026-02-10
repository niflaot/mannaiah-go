package contacts

import (
	"context"
	"errors"
	stdhttp "net/http"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"gorm.io/gorm"
	"mannaiah/module/contacts/adapter/store"
	"mannaiah/module/contacts/port"
	coredb "mannaiah/module/core/database"
	corehttp "mannaiah/module/core/http"
)

// TestNewAndRegisterRoutes verifies module wiring and route registration.
func TestNewAndRegisterRoutes(t *testing.T) {
	db := newDBForTest(t)
	module, err := New(db)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8101}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(module.RegisterRoutes)

	req, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/contacts?page=1&limit=10", nil)
	resp, testErr := server.App().Test(req)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if resp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, stdhttp.StatusOK)
	}
}

// TestRegisterRoutesNilModule verifies nil module route registration behavior.
func TestRegisterRoutesNilModule(t *testing.T) {
	var module *Module
	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8102}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}

	server.RegisterRoutes(module.RegisterRoutes)
}

// TestNewRejectsNilDB verifies module constructor validation for nil DB dependencies.
func TestNewRejectsNilDB(t *testing.T) {
	_, err := New(nil)
	if !errors.Is(err, store.ErrNilDB) {
		t.Fatalf("New() error = %v, want ErrNilDB", err)
	}
}

// integrationEventPublisherProbe defines integration publish behavior for module tests.
type integrationEventPublisherProbe struct{}

// Publish implements integration event publisher contracts.
func (integrationEventPublisherProbe) Publish(ctx context.Context, event port.IntegrationEvent) error {
	return nil
}

// TestResolvePublisher verifies optional publisher resolution behavior.
func TestResolvePublisher(t *testing.T) {
	if value := resolvePublisher(nil); value != nil {
		t.Fatalf("expected nil publisher on empty values")
	}

	first := integrationEventPublisherProbe{}
	second := integrationEventPublisherProbe{}
	value := resolvePublisher([]port.IntegrationEventPublisher{first, second})
	if value == nil {
		t.Fatalf("expected resolved publisher")
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
	module, err := New(db)
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
	module, err := New(db)
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
	module, err := New(db)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := module.Load(nil); err != nil {
		t.Fatalf("Load(nil) error = %v", err)
	}
}

// newDBForTest creates an in-memory DB for module tests.
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
