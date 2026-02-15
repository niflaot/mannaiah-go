package runtime

import (
	"context"
	"errors"
	stdhttp "net/http"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"gorm.io/gorm"
	coredb "mannaiah/module/core/database"
	corehttp "mannaiah/module/core/http"
	ordersstore "mannaiah/module/orders/adapter/store"
	ordersapplication "mannaiah/module/orders/application"
	ordersport "mannaiah/module/orders/port"
)

// customerSourceProbe defines customer-source behavior for module tests.
type customerSourceProbe struct{}

// GetByID returns fixed customer values.
func (customerSourceProbe) GetByID(ctx context.Context, id string) (*ordersport.Customer, error) {
	return &ordersport.Customer{ID: id, Address: "A", CityCode: "11001"}, nil
}

// TestNewAndRegisterRoutes verifies module wiring and route registration.
func TestNewAndRegisterRoutes(t *testing.T) {
	db := newDBForTest(t)
	module, err := New(db, customerSourceProbe{}, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8181}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(module.RegisterRoutes)

	req, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/orders?page=1&limit=10", nil)
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
	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8182}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}

	server.RegisterRoutes(module.RegisterRoutes)
}

// TestNewValidation verifies constructor validation behavior.
func TestNewValidation(t *testing.T) {
	db := newDBForTest(t)

	if _, err := New(nil, customerSourceProbe{}, nil); !errors.Is(err, ordersstore.ErrNilDB) {
		t.Fatalf("New(nil) error = %v, want ErrNilDB", err)
	}
	if _, err := New(db, nil, nil); !errors.Is(err, ordersapplication.ErrNilCustomerSource) {
		t.Fatalf("New(nil customer source) error = %v, want ErrNilCustomerSource", err)
	}
}

// loaderProbe defines startup loader behavior for module tests.
type loaderProbe struct {
	// registered reports whether routes were registered.
	registered bool
	// specAdded reports whether OpenAPI specs were added.
	specAdded bool
}

// RegisterRoutes captures route registration calls.
func (l *loaderProbe) RegisterRoutes(register func(router corehttp.Router)) {
	l.registered = true
}

// AddOpenAPISpec captures OpenAPI merge calls.
func (l *loaderProbe) AddOpenAPISpec(spec *openapi3.T) error {
	l.specAdded = spec != nil
	return nil
}

// TestModuleLoad verifies module self-loading behavior for routes and OpenAPI specs.
func TestModuleLoad(t *testing.T) {
	db := newDBForTest(t)
	module, err := New(db, customerSourceProbe{}, nil)
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

// RegisterRoutes ignores route registration calls.
func (failingLoaderProbe) RegisterRoutes(register func(router corehttp.Router)) {}

// AddOpenAPISpec returns forced merge errors.
func (failingLoaderProbe) AddOpenAPISpec(spec *openapi3.T) error {
	return errors.New("merge failed")
}

// TestModuleLoadError verifies loader merge failures are returned.
func TestModuleLoadError(t *testing.T) {
	db := newDBForTest(t)
	module, err := New(db, customerSourceProbe{}, nil)
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
	module, err := New(db, customerSourceProbe{}, nil)
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
	module, err := New(db, customerSourceProbe{}, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	module.SetAuthorizer(nil)
}

// newDBForTest creates an in-memory DB for module tests.
func newDBForTest(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := coredb.Open(coredb.Config{Driver: "sqlite", DSN: "file::memory:?cache=shared", MaxOpenConns: 1}, nil)
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
