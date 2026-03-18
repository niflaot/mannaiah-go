package runtime

import (
	"bytes"
	"context"
	stdhttp "net/http"
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

// authorizerMock defines authorizer behavior for module tests.
type authorizerMock struct{}

// Require authenticates requests.
func (authorizerMock) Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error {
	return nil
}

// IsUnauthorized reports auth failures.
func (authorizerMock) IsUnauthorized(err error) bool {
	return false
}

// IsForbidden reports authorization failures.
func (authorizerMock) IsForbidden(err error) bool {
	return false
}

// TestLoadRegisterRoutes verifies module route/spec registration behavior.
func TestLoadRegisterRoutes(t *testing.T) {
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
		t.Fatalf("expected spec merge")
	}
}

// TestRegisterRoutesUnavailable verifies documented-but-unavailable integration behavior.
func TestRegisterRoutesUnavailable(t *testing.T) {
	module, err := New(Config{
		Enabled:             true,
		TCCBaseURL:          "https://testsomos.tcc.com.co",
		TCCAccessToken:      "",
		TCCIdentifier:       "",
		TCCAccount:          "7000880",
		TCCRequestTimeoutMS: 1000,
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8401}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(module.RegisterRoutes)

	request, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/shipping/quotes", bytes.NewBufferString(`{"carrier":"tcc","businessUnit":"courier","originCityCode":"05001","destinationCityCode":"11001","declaredValue":120000,"units":[{"number":1,"realWeight":2.1,"height":15,"width":20,"length":30}]}`))
	request.Header.Set("Content-Type", "application/json")

	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != stdhttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.StatusCode, stdhttp.StatusServiceUnavailable)
	}
}

// TestRegisterRoutesSuccess verifies configured integration route behavior.
func TestRegisterRoutesSuccess(t *testing.T) {
	upstream := httptest.NewServer(stdhttp.HandlerFunc(func(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"codigoResultado":"0","mensajeResultado":"OK","total":{"totaldespacho":48000,"unidadnegocio":"PAQUETERIA"}}`))
	}))
	defer upstream.Close()

	module, err := New(Config{
		Enabled:                  true,
		TCCBaseURL:               upstream.URL,
		TCCAccessToken:           "token",
		TCCAccount:               "7000880",
		TCCIdentifier:            "901599500",
		TCCRequestTimeoutMS:      3000,
		TCCCircuitBreakerEnabled: false,
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	module.SetAuthorizer(authorizerMock{})

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8402}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(module.RegisterRoutes)

	request, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/shipping/quotes", bytes.NewBufferString(`{"carrier":"tcc","businessUnit":"courier","originCityCode":"05001","destinationCityCode":"11001","declaredValue":120000,"units":[{"number":1,"realWeight":2.1,"height":15,"width":20,"length":30}]}`))
	request.Header.Set("Content-Type", "application/json")

	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != stdhttp.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, stdhttp.StatusOK)
	}
}
