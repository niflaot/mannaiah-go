package http

import (
	"context"
	errorspkg "errors"
	stdhttp "net/http"
	"strings"
	"testing"
	"time"

	corehttp "mannaiah/module/core/http"
	brandservice "mannaiah/module/falabella/application/brand/service"
	productsyncservice "mannaiah/module/falabella/application/productsync/service"
	syncstatusservice "mannaiah/module/falabella/application/syncstatus/service"
	syncdomain "mannaiah/module/falabella/domain/sync"
	"mannaiah/module/falabella/port"
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
	handler, err := NewHandler(&serviceMock{payload: []byte(`{"ok":true}`)}, &productSyncServiceMock{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	handler.SetAuthorizer(&authorizerMock{requireErr: errUnauthorized})

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

// syncStatusServiceMock defines sync status service behavior for handler tests.
type syncStatusServiceMock struct {
	// entry defines GetByFeedID() return values.
	entry *syncdomain.SyncEntry
	// entries defines GetByProductID() return values.
	entries []syncdomain.SyncEntry
	// resolveResult defines ResolveFeedStatus() return values.
	resolveResult *syncstatusservice.ResolveResult
	// err defines service errors.
	err error
}

// GetByFeedID returns configured entry/error values.
func (m *syncStatusServiceMock) GetByFeedID(ctx context.Context, feedID string) (*syncdomain.SyncEntry, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.entry, nil
}

// GetByProductID returns configured entries/error values.
func (m *syncStatusServiceMock) GetByProductID(ctx context.Context, productID string) ([]syncdomain.SyncEntry, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.entries, nil
}

// ResolveFeedStatus returns configured result/error values.
func (m *syncStatusServiceMock) ResolveFeedStatus(ctx context.Context, feedID string) (*syncstatusservice.ResolveResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.resolveResult, nil
}

// TestGetSyncStatusByFeedRoute verifies feed status lookup route behavior.
func TestGetSyncStatusByFeedRoute(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	statusSvc := &syncStatusServiceMock{entry: &syncdomain.SyncEntry{
		ProductID: "prod-1", SKU: "SKU-001", FeedID: "feed-abc",
		Action: syncdomain.SyncActionCreate, Status: syncdomain.SyncStatusPending, SyncedAt: now,
	}}
	handler, err := NewHandler(&serviceMock{payload: []byte(`{"ok":true}`)}, &productSyncServiceMock{}, statusSvc)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server, _ := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8301}, nil)
	server.RegisterRoutes(handler.RegisterRoutes)

	request, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/falabella/sync/status/feed/feed-abc", nil)
	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != stdhttp.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, stdhttp.StatusOK)
	}
}

// TestGetSyncStatusByProductRoute verifies product status lookup route behavior.
func TestGetSyncStatusByProductRoute(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	statusSvc := &syncStatusServiceMock{entries: []syncdomain.SyncEntry{
		{ProductID: "prod-1", SKU: "SKU-001", FeedID: "feed-abc",
			Action: syncdomain.SyncActionCreate, Status: syncdomain.SyncStatusPending, SyncedAt: now},
	}}
	handler, err := NewHandler(&serviceMock{payload: []byte(`{"ok":true}`)}, &productSyncServiceMock{}, statusSvc)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server, _ := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8302}, nil)
	server.RegisterRoutes(handler.RegisterRoutes)

	request, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/falabella/sync/status/product/prod-1", nil)
	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != stdhttp.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, stdhttp.StatusOK)
	}
}

// TestResolveFeedStatusRoute verifies feed resolution route behavior.
func TestResolveFeedStatusRoute(t *testing.T) {
	statusSvc := &syncStatusServiceMock{resolveResult: &syncstatusservice.ResolveResult{
		FeedID: "feed-abc", Status: "Finished", TotalRecords: 1, ProcessedRecords: 1,
	}}
	handler, err := NewHandler(&serviceMock{payload: []byte(`{"ok":true}`)}, &productSyncServiceMock{}, statusSvc)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server, _ := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8303}, nil)
	server.RegisterRoutes(handler.RegisterRoutes)

	request, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/falabella/sync/status/feed/feed-abc/resolve", nil)
	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != stdhttp.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, stdhttp.StatusOK)
	}
}

// TestSyncStatusRouteNotConfigured verifies 503 when sync status service is nil.
func TestSyncStatusRouteNotConfigured(t *testing.T) {
	handler, err := NewHandler(&serviceMock{payload: []byte(`{"ok":true}`)}, &productSyncServiceMock{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server, _ := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8304}, nil)
	server.RegisterRoutes(handler.RegisterRoutes)

	request, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/falabella/sync/status/feed/feed-abc", nil)
	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != stdhttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.StatusCode, stdhttp.StatusServiceUnavailable)
	}
}

// TestMapErrorSyncStatus verifies sync status error mapping behavior.
func TestMapErrorSyncStatus(t *testing.T) {
	handler, _ := NewHandler(&serviceMock{payload: []byte(`{"ok":true}`)}, &productSyncServiceMock{})

	if appErr := handler.mapError(syncstatusservice.ErrInvalidFeedID); appErr == nil {
		t.Fatalf("expected invalid feed id mapping")
	}
	if appErr := handler.mapError(syncstatusservice.ErrFeedNotFinished); appErr == nil {
		t.Fatalf("expected feed not finished mapping")
	}
	if appErr := handler.mapError(port.ErrSyncEntryNotFound); appErr == nil {
		t.Fatalf("expected sync entry not found mapping")
	}
}