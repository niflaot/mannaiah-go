package http

import (
	"context"
	errorspkg "errors"
	stdhttp "net/http"
	"testing"

	corehttp "mannaiah/module/core/http"
	woocontactservice "mannaiah/module/woocommerce/application/contact/service"
)

// serviceMock defines WooCommerce service behavior for handler tests.
type serviceMock struct {
	// summary defines sync summary responses.
	summary *woocontactservice.SyncSummary
	// syncErr defines sync execution errors.
	syncErr error
}

// ValidateIntegration validates integration state.
func (m *serviceMock) ValidateIntegration(ctx context.Context) error {
	return nil
}

// SyncContacts performs sync behavior.
func (m *serviceMock) SyncContacts(ctx context.Context, trigger string) (*woocontactservice.SyncSummary, error) {
	if m.syncErr != nil {
		return nil, m.syncErr
	}
	if m.summary != nil {
		return m.summary, nil
	}

	return &woocontactservice.SyncSummary{Trigger: trigger}, nil
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
	if _, err := NewHandler(nil); !errorspkg.Is(err, ErrNilService) {
		t.Fatalf("NewHandler(nil) error = %v, want ErrNilService", err)
	}
}

// TestRegisterRoutesAndSync verifies route registration and successful sync behavior.
func TestRegisterRoutesAndSync(t *testing.T) {
	handler, err := NewHandler(&serviceMock{
		summary: &woocontactservice.SyncSummary{Trigger: "manual", Processed: 2},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8121}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(handler.RegisterRoutes)

	request, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/woo/sync/contacts", nil)
	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != stdhttp.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, stdhttp.StatusOK)
	}
}

// TestRegisterRoutesWithAuth verifies protected route behavior.
func TestRegisterRoutesWithAuth(t *testing.T) {
	handler, err := NewHandler(&serviceMock{}, &authorizerMock{requireErr: errUnauthorized})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8122}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(handler.RegisterRoutes)

	request, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/woo/sync/contacts", nil)
	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != stdhttp.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.StatusCode, stdhttp.StatusUnauthorized)
	}
}

// TestMapError verifies sync error mapping behavior.
func TestMapError(t *testing.T) {
	handler, err := NewHandler(&serviceMock{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	if appErr := handler.mapError(woocontactservice.ErrSyncDisabled); appErr == nil {
		t.Fatalf("expected mapError(sync disabled)")
	}
	if appErr := handler.mapError(woocontactservice.ErrIntegrationUnavailable); appErr == nil {
		t.Fatalf("expected mapError(integration unavailable)")
	}
	if appErr := handler.mapError(errorspkg.New("unknown")); appErr == nil {
		t.Fatalf("expected mapError(unknown)")
	}
}

// TestSetAuthorizer verifies optional authorizer wiring behavior.
func TestSetAuthorizer(t *testing.T) {
	handler, err := NewHandler(&serviceMock{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	handler.SetAuthorizer(nil)
}
