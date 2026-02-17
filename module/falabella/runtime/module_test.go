package runtime

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"go.uber.org/zap"
	corehttp "mannaiah/module/core/http"
)

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

// failingLoaderProbe defines loader behavior that fails while merging specs.
type failingLoaderProbe struct{}

// RegisterRoutes ignores registration calls.
func (failingLoaderProbe) RegisterRoutes(register func(router corehttp.Router)) {}

// AddOpenAPISpec returns a forced merge error.
func (failingLoaderProbe) AddOpenAPISpec(spec *openapi3.T) error {
	return errors.New("merge failed")
}

// TestNewWithInvalidConfigKeepsRoute verifies invalid-config fallback behavior.
func TestNewWithInvalidConfigKeepsRoute(t *testing.T) {
	module, err := New(Config{}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8194}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(module.RegisterRoutes)

	request, _ := http.NewRequest(http.MethodGet, "/falabella/brands", nil)
	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusServiceUnavailable)
	}

	syncRequest, _ := http.NewRequest(http.MethodPost, "/falabella/sync/products/p-1", nil)
	syncResponse, syncErr := server.App().Test(syncRequest)
	if syncErr != nil {
		t.Fatalf("App().Test(sync) error = %v", syncErr)
	}
	if syncResponse.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("sync status = %d, want %d", syncResponse.StatusCode, http.StatusServiceUnavailable)
	}
}

// TestNewWithValidConfig verifies valid integration behavior.
func TestNewWithValidConfig(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"SuccessResponse":{"Head":{"RequestId":"r1"}}}`))
	}))
	defer upstream.Close()

	module, err := New(Config{
		URL:     upstream.URL,
		UserID:  "user-1",
		APIKey:  "key-1",
		Version: "1.0",
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8195}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(module.RegisterRoutes)

	request, _ := http.NewRequest(http.MethodGet, "/falabella/brands", nil)
	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
}

// TestModuleLoad verifies module self-loading behavior.
func TestModuleLoad(t *testing.T) {
	module, err := New(Config{}, zap.NewNop())
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

// TestModuleLoadError verifies loader merge failures are returned.
func TestModuleLoadError(t *testing.T) {
	module, err := New(Config{}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := module.Load(failingLoaderProbe{}); err == nil {
		t.Fatalf("expected load error")
	}
}

// TestSetAuthorizer verifies optional authorizer wiring behavior.
func TestSetAuthorizer(t *testing.T) {
	module, err := New(Config{}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	module.SetAuthorizer(nil)
}
