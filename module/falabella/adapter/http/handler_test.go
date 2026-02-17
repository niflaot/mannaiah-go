package http

import (
	"context"
	errorspkg "errors"
	stdhttp "net/http"
	"strings"
	"testing"

	corehttp "mannaiah/module/core/http"
	brandservice "mannaiah/module/falabella/application/brand/service"
	productsyncservice "mannaiah/module/falabella/application/productsync/service"
)

// serviceMock defines Falabella brand service behavior for handler tests.
type serviceMock struct {
	// payload defines successful payload values.
	payload []byte
	// err defines service execution errors.
	err error
}

// GetBrands returns configured payload/error values.
func (m *serviceMock) GetBrands(ctx context.Context) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.payload, nil
}

// productSyncServiceMock defines product-sync behavior for handler tests.
type productSyncServiceMock struct {
	// summary defines sync-summary return values.
	summary *productsyncservice.Summary
	// err defines sync-service errors.
	err error
	// ids captures list sync identifiers.
	ids []string
	// id captures single sync identifiers.
	id string
}

// SyncProduct returns configured summary/error values.
func (m *productSyncServiceMock) SyncProduct(ctx context.Context, id string) (*productsyncservice.Summary, error) {
	m.id = id
	if m.err != nil {
		return nil, m.err
	}

	return m.summary, nil
}

// SyncProducts returns configured summary/error values.
func (m *productSyncServiceMock) SyncProducts(ctx context.Context, ids []string) (*productsyncservice.Summary, error) {
	m.ids = append([]string(nil), ids...)
	if m.err != nil {
		return nil, m.err
	}

	return m.summary, nil
}

// authorizerMock defines authorization behavior for handler tests.
type authorizerMock struct {
	// requireErr defines auth errors.
	requireErr error
}

// Require authenticates and authorizes requests.
func (m *authorizerMock) Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error {
	return m.requireErr
}

// IsUnauthorized reports unauthorized errors.
func (m *authorizerMock) IsUnauthorized(err error) bool {
	return errorspkg.Is(err, errUnauthorized)
}

// IsForbidden reports forbidden errors.
func (m *authorizerMock) IsForbidden(err error) bool {
	return errorspkg.Is(err, errForbidden)
}

var (
	// errUnauthorized defines unauthorized test errors.
	errUnauthorized = errorspkg.New("unauthorized")
	// errForbidden defines forbidden test errors.
	errForbidden = errorspkg.New("forbidden")
)

// TestNewHandlerValidation verifies constructor validation behavior.
func TestNewHandlerValidation(t *testing.T) {
	_, err := NewHandler(nil, &productSyncServiceMock{})
	if !errorspkg.Is(err, ErrNilService) {
		t.Fatalf("NewHandler() error = %v, want %v", err, ErrNilService)
	}
	_, err = NewHandler(&serviceMock{payload: []byte(`{}`)}, nil)
	if !errorspkg.Is(err, ErrNilProductSyncService) {
		t.Fatalf("NewHandler() error = %v, want %v", err, ErrNilProductSyncService)
	}
}

// TestGetBrandsRoute verifies route registration and successful behavior.
func TestGetBrandsRoute(t *testing.T) {
	handler, err := NewHandler(&serviceMock{payload: []byte(`{"SuccessResponse":{}}`)}, &productSyncServiceMock{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8191}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(handler.RegisterRoutes)

	request, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/falabella/brands", nil)
	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != stdhttp.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, stdhttp.StatusOK)
	}
}

// TestGetBrandsRouteInvalidPayload verifies invalid payload behavior.
func TestGetBrandsRouteInvalidPayload(t *testing.T) {
	handler, err := NewHandler(&serviceMock{payload: []byte("not-json")}, &productSyncServiceMock{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8192}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(handler.RegisterRoutes)

	request, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/falabella/brands", nil)
	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != stdhttp.StatusBadGateway {
		t.Fatalf("status = %d, want %d", response.StatusCode, stdhttp.StatusBadGateway)
	}
}

// TestGetBrandsRouteWithAuth verifies protected route behavior.
func TestGetBrandsRouteWithAuth(t *testing.T) {
	handler, err := NewHandler(&serviceMock{payload: []byte(`{"ok":true}`)}, &productSyncServiceMock{}, &authorizerMock{requireErr: errUnauthorized})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8193}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(handler.RegisterRoutes)

	request, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/falabella/brands", nil)
	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != stdhttp.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.StatusCode, stdhttp.StatusUnauthorized)
	}
}

// TestMapError verifies Falabella error mapping behavior.
func TestMapError(t *testing.T) {
	handler, err := NewHandler(&serviceMock{payload: []byte(`{"ok":true}`)}, &productSyncServiceMock{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	if appErr := handler.mapError(brandservice.ErrIntegrationUnavailable); appErr == nil {
		t.Fatalf("expected integration unavailable mapping")
	}
	if appErr := handler.mapError(productsyncservice.ErrIntegrationUnavailable); appErr == nil {
		t.Fatalf("expected integration unavailable mapping")
	}
	if appErr := handler.mapError(productsyncservice.ErrInvalidProductID); appErr == nil {
		t.Fatalf("expected invalid product id mapping")
	}
	if appErr := handler.mapError(errorspkg.New("boom")); appErr == nil {
		t.Fatalf("expected unknown mapping")
	}
}

// TestSetAuthorizer verifies optional authorizer wiring behavior.
func TestSetAuthorizer(t *testing.T) {
	handler, err := NewHandler(&serviceMock{payload: []byte(`{"ok":true}`)}, &productSyncServiceMock{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	handler.SetAuthorizer(nil)
}

// TestSyncProductsRoute verifies batch sync route behavior.
func TestSyncProductsRoute(t *testing.T) {
	syncService := &productSyncServiceMock{summary: &productsyncservice.Summary{Requested: 2, Synced: 2}}
	handler, err := NewHandler(&serviceMock{payload: []byte(`{"ok":true}`)}, syncService)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8196}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(handler.RegisterRoutes)

	request, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/falabella/sync/products", strings.NewReader(`{"ids":["p-1","p-2"]}`))
	request.Header.Set("Content-Type", "application/json")
	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != stdhttp.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, stdhttp.StatusOK)
	}
	if len(syncService.ids) != 2 {
		t.Fatalf("len(syncService.ids) = %d, want %d", len(syncService.ids), 2)
	}
}

// TestSyncProductByIDRoute verifies single sync route behavior.
func TestSyncProductByIDRoute(t *testing.T) {
	syncService := &productSyncServiceMock{summary: &productsyncservice.Summary{Requested: 1, Synced: 1}}
	handler, err := NewHandler(&serviceMock{payload: []byte(`{"ok":true}`)}, syncService)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8197}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(handler.RegisterRoutes)

	request, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/falabella/sync/products/p-1", nil)
	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != stdhttp.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, stdhttp.StatusOK)
	}
	if syncService.id != "p-1" {
		t.Fatalf("syncService.id = %q, want %q", syncService.id, "p-1")
	}
}

// TestSyncProductsRouteInvalidBody verifies invalid-body behavior.
func TestSyncProductsRouteInvalidBody(t *testing.T) {
	handler, err := NewHandler(&serviceMock{payload: []byte(`{"ok":true}`)}, &productSyncServiceMock{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8198}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(handler.RegisterRoutes)

	request, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/falabella/sync/products", strings.NewReader("{invalid"))
	request.Header.Set("Content-Type", "application/json")
	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != stdhttp.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.StatusCode, stdhttp.StatusBadRequest)
	}
}

// TestSyncProductsRouteWithoutBody verifies empty-body batch sync behavior.
func TestSyncProductsRouteWithoutBody(t *testing.T) {
	syncService := &productSyncServiceMock{summary: &productsyncservice.Summary{Requested: 1, Synced: 1}}
	handler, err := NewHandler(&serviceMock{payload: []byte(`{"ok":true}`)}, syncService)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8199}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(handler.RegisterRoutes)

	request, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/falabella/sync/products", nil)
	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != stdhttp.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, stdhttp.StatusOK)
	}
	if syncService.ids != nil {
		t.Fatalf("syncService.ids = %v, want nil", syncService.ids)
	}
}
