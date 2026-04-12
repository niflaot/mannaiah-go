package storefront

import (
	"context"
	"errors"
	stdhttp "net/http"
	"testing"

	corehttp "mannaiah/module/core/http"
	storefrontdomain "mannaiah/module/products/domain/storefront"
)

// serviceMock defines storefront navigation service behavior for tests.
type serviceMock struct {
	// getFn defines navigation retrieval behavior.
	getFn func(ctx context.Context) (*storefrontdomain.Navigation, error)
}

// Get executes configured navigation retrieval behavior.
func (m serviceMock) Get(ctx context.Context) (*storefrontdomain.Navigation, error) {
	return m.getFn(ctx)
}

// Regenerate is unused in handler tests.
func (m serviceMock) Regenerate(ctx context.Context) (*storefrontdomain.Navigation, error) {
	return nil, nil
}

// TriggerRefresh is unused in handler tests.
func (m serviceMock) TriggerRefresh(ctx context.Context) {}

// authorizerMock defines storefront endpoint auth behavior for tests.
type authorizerMock struct {
	// requireFn defines auth behavior.
	requireFn func(ctx context.Context, header string, requiredPermissions ...string) error
}

// Require executes configured auth behavior.
func (m authorizerMock) Require(ctx context.Context, header string, requiredPermissions ...string) error {
	if m.requireFn != nil {
		return m.requireFn(ctx, header, requiredPermissions...)
	}

	return nil
}

// IsUnauthorized reports authentication failures for tests.
func (m authorizerMock) IsUnauthorized(err error) bool {
	return errors.Is(err, errUnauthorized)
}

// IsForbidden reports authorization failures for tests.
func (m authorizerMock) IsForbidden(err error) bool {
	return errors.Is(err, errForbidden)
}

var (
	// errUnauthorized defines authentication failures for tests.
	errUnauthorized = errors.New("unauthorized")
	// errForbidden defines authorization failures for tests.
	errForbidden = errors.New("forbidden")
)

// TestRegisterRoutesReturnsNavigation verifies protected navigation route behavior.
func TestRegisterRoutesReturnsNavigation(t *testing.T) {
	handler, err := NewHandler(serviceMock{
		getFn: func(ctx context.Context) (*storefrontdomain.Navigation, error) {
			return &storefrontdomain.Navigation{Realm: "default"}, nil
		},
	}, authorizerMock{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8123}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(handler.RegisterRoutes)

	req, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/storefront/navigation", nil)
	resp, testErr := server.App().Test(req)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if resp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, stdhttp.StatusOK)
	}
}

// TestRegisterRoutesRequiresPermission verifies auth failures are mapped.
func TestRegisterRoutesRequiresPermission(t *testing.T) {
	handler, err := NewHandler(serviceMock{
		getFn: func(ctx context.Context) (*storefrontdomain.Navigation, error) {
			return &storefrontdomain.Navigation{Realm: "default"}, nil
		},
	}, authorizerMock{
		requireFn: func(ctx context.Context, header string, requiredPermissions ...string) error {
			if len(requiredPermissions) != 1 || requiredPermissions[0] != "storefront:manage" {
				t.Fatalf("requiredPermissions = %#v, want storefront:manage", requiredPermissions)
			}
			return errForbidden
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8124}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(handler.RegisterRoutes)

	req, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/storefront/navigation", nil)
	resp, testErr := server.App().Test(req)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if resp.StatusCode != stdhttp.StatusForbidden {
		t.Fatalf("status = %d, want %d", resp.StatusCode, stdhttp.StatusForbidden)
	}
}
