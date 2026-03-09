package runtime

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"testing"

	coredbmigration "mannaiah/module/core/database/migration"
	corehttp "mannaiah/module/core/http"

	"github.com/getkin/kin-openapi/openapi3"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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

// TestNewWithValidConfig verifies valid integration behavior without startup API calls.
func TestNewWithValidConfig(t *testing.T) {
	requestCount := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount++
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

	if requestCount != 0 {
		t.Fatalf("startup made %d outbound requests, want 0 (GetBrands should not be called at startup)", requestCount)
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

// TestConfigureSyncStatusNilDB verifies nil DB is silently ignored.
func TestConfigureSyncStatusNilDB(t *testing.T) {
	module, err := New(Config{}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if configErr := module.ConfigureSyncStatus(nil); configErr != nil {
		t.Fatalf("ConfigureSyncStatus(nil) error = %v", configErr)
	}
}

// TestConfigureSyncStatusPreservesImageTranscode verifies sync-status wiring keeps image-transcode endpoint behavior.
func TestConfigureSyncStatusPreservesImageTranscode(t *testing.T) {
	sourceImage := image.NewRGBA(image.Rect(0, 0, 2, 2))
	sourceImage.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	sourceBuffer := bytes.Buffer{}
	if err := png.Encode(&sourceBuffer, sourceImage); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}

	sourceServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_ = request
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(sourceBuffer.Bytes())
	}))
	defer sourceServer.Close()

	module, err := New(Config{
		URL:                                  sourceServer.URL,
		UserID:                               "user-1",
		APIKey:                               "key-1",
		ProductImageTranscodeEnabled:         true,
		ProductImageTranscodeAllowedPrefixes: sourceServer.URL,
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}
	if err := coredbmigration.Apply(context.Background(), db, coredbmigration.Config{Enabled: true, Driver: "sqlite", Table: "schema_migrations"}, nil); err != nil {
		t.Fatalf("migration.Apply() error = %v", err)
	}
	if err := module.ConfigureSyncStatus(db); err != nil {
		t.Fatalf("ConfigureSyncStatus() error = %v", err)
	}

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8307}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(module.RegisterRoutes)

	request, _ := http.NewRequest(
		http.MethodGet,
		"/falabella/images/transcoded?src="+neturl.QueryEscape(sourceServer.URL+"/sample.png"),
		nil,
	)
	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	body := bytes.Buffer{}
	if _, readErr := body.ReadFrom(response.Body); readErr != nil {
		t.Fatalf("ReadFrom(response body) error = %v", readErr)
	}
	if _, decodeErr := jpeg.Decode(bytes.NewReader(body.Bytes())); decodeErr != nil {
		t.Fatalf("jpeg.Decode() error = %v", decodeErr)
	}
}

// TestSyncStatusRoutesWithoutDB verifies sync status returns 503 without DB.
func TestSyncStatusRoutesWithoutDB(t *testing.T) {
	module, err := New(Config{}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	server, _ := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8305}, nil)
	server.RegisterRoutes(module.RegisterRoutes)

	request, _ := http.NewRequest(http.MethodGet, "/falabella/sync/status/feed/feed-abc", nil)
	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusServiceUnavailable)
	}
}
