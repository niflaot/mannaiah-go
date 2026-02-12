package http

import (
	"context"
	errorspkg "errors"
	"net/http"
	"testing"

	"mannaiah/module/auth/application"
	corehttp "mannaiah/module/core/http"
)

// serviceMock defines auth behavior for handler tests.
type serviceMock struct {
	// requireFn defines require behavior.
	requireFn func(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
}

// Require executes configured require behavior.
func (m serviceMock) Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error {
	return m.requireFn(ctx, authorizationHeader, requiredPermissions...)
}

// TestNewHandlerRejectsNilService verifies constructor validation behavior.
func TestNewHandlerRejectsNilService(t *testing.T) {
	if _, err := NewHandler(nil); !errorspkg.Is(err, ErrNilService) {
		t.Fatalf("NewHandler(nil) error = %v, want ErrNilService", err)
	}
}

// TestCheckAuthSuccess verifies successful authentication behavior.
func TestCheckAuthSuccess(t *testing.T) {
	handler, err := NewHandler(serviceMock{requireFn: func(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error {
		return nil
	}})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := newHTTPServerForHandler(t, handler)
	request, _ := http.NewRequest(http.MethodGet, "/check-auth", nil)
	request.Header.Set("Authorization", "Bearer token")
	response := runRequest(t, server, request)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
}

// TestCheckAuthUnauthorized verifies unauthorized mapping behavior.
func TestCheckAuthUnauthorized(t *testing.T) {
	handler, err := NewHandler(serviceMock{requireFn: func(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error {
		return application.ErrUnauthorized
	}})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := newHTTPServerForHandler(t, handler)
	request, _ := http.NewRequest(http.MethodGet, "/check-auth", nil)
	response := runRequest(t, server, request)
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusUnauthorized)
	}
}

// TestMapErrorVariants verifies mapped error behavior.
func TestMapErrorVariants(t *testing.T) {
	handler := &Handler{}
	if mapped := handler.mapError(application.ErrUnauthorized); mapped == nil {
		t.Fatalf("expected unauthorized mapped error")
	}
	if mapped := handler.mapError(application.ErrForbidden); mapped == nil {
		t.Fatalf("expected forbidden mapped error")
	}
	if mapped := handler.mapError(errorspkg.New("boom")); mapped == nil {
		t.Fatalf("expected internal mapped error")
	}
}

// newHTTPServerForHandler creates servers for handler tests.
func newHTTPServerForHandler(t *testing.T, handler *Handler) *corehttp.Server {
	t.Helper()

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8160}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(handler.RegisterRoutes)

	return server
}

// runRequest runs HTTP requests against test servers.
func runRequest(t *testing.T, server *corehttp.Server, request *http.Request) *http.Response {
	t.Helper()

	response, err := server.App().Test(request)
	if err != nil {
		t.Fatalf("App().Test() error = %v", err)
	}

	return response
}
