package application

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"mannaiah/module/auth/domain"
)

// verifierMock defines token verifier behavior for auth-service tests.
type verifierMock struct {
	// verifyFn defines configurable token verification behavior.
	verifyFn func(ctx context.Context, token string) (*domain.Claims, error)
}

// Verify executes configured token verification behavior.
func (m verifierMock) Verify(ctx context.Context, token string) (*domain.Claims, error) {
	return m.verifyFn(ctx, token)
}

// TestNewServiceRejectsNilVerifier verifies service constructor validation behavior.
func TestNewServiceRejectsNilVerifier(t *testing.T) {
	_, err := NewService("development", "", "", nil, zap.NewNop())
	if !errors.Is(err, ErrNilVerifier) {
		t.Fatalf("NewService() error = %v, want ErrNilVerifier", err)
	}
}

// TestRequireAllowsExactPermission verifies exact scope matching behavior.
func TestRequireAllowsExactPermission(t *testing.T) {
	service := newServiceForTest(t, "development", "", "", verifierMock{
		verifyFn: func(ctx context.Context, token string) (*domain.Claims, error) {
			if token != "jwt-token" {
				t.Fatalf("token = %q, want %q", token, "jwt-token")
			}
			return &domain.Claims{Subject: "user-1", Scope: "contacts:read contacts:create"}, nil
		},
	})

	err := service.Require(context.Background(), "Bearer jwt-token", "contacts:create")
	if err != nil {
		t.Fatalf("Require() error = %v", err)
	}
}

// TestRequireAllowsManagePermission verifies manage wildcard matching behavior.
func TestRequireAllowsManagePermission(t *testing.T) {
	service := newServiceForTest(t, "development", "", "", verifierMock{
		verifyFn: func(ctx context.Context, token string) (*domain.Claims, error) {
			return &domain.Claims{Subject: "user-1", Scope: "contacts:manage"}, nil
		},
	})

	err := service.Require(context.Background(), "Bearer jwt-token", "contacts:update")
	if err != nil {
		t.Fatalf("Require() error = %v", err)
	}
}

// TestRequireRejectsMissingScope verifies denied authorization on missing scope values.
func TestRequireRejectsMissingScope(t *testing.T) {
	service := newServiceForTest(t, "development", "", "", verifierMock{
		verifyFn: func(ctx context.Context, token string) (*domain.Claims, error) {
			return &domain.Claims{Subject: "user-1", Scope: ""}, nil
		},
	})

	err := service.Require(context.Background(), "Bearer jwt-token", "contacts:read")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Require() error = %v, want ErrForbidden", err)
	}
}

// TestRequireRejectsInvalidBearer verifies invalid authorization header behavior.
func TestRequireRejectsInvalidBearer(t *testing.T) {
	service := newServiceForTest(t, "development", "", "", verifierMock{
		verifyFn: func(ctx context.Context, token string) (*domain.Claims, error) {
			return &domain.Claims{Subject: "user-1", Scope: "contacts:read"}, nil
		},
	})

	err := service.Require(context.Background(), "invalid", "contacts:read")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Require() error = %v, want ErrUnauthorized", err)
	}
}

// TestAuthenticateUsesDevBypass verifies development bypass behavior and debug logging.
func TestAuthenticateUsesDevBypass(t *testing.T) {
	core, observed := observer.New(zap.DebugLevel)
	logger := zap.New(core)
	invocations := 0

	service := newServiceForTest(t, "development", "dev-token", "", verifierMock{
		verifyFn: func(ctx context.Context, token string) (*domain.Claims, error) {
			invocations++
			return nil, errors.New("unexpected call")
		},
	}, logger)

	claims, err := service.Authenticate(context.Background(), "Bearer dev-token")
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if claims.Subject != "dev-admin" {
		t.Fatalf("subject = %q, want %q", claims.Subject, "dev-admin")
	}
	if invocations != 0 {
		t.Fatalf("verifier invocations = %d, want %d", invocations, 0)
	}
	if observed.Len() != 1 {
		t.Fatalf("logs = %d, want %d", observed.Len(), 1)
	}
	if observed.All()[0].Message != "Using Dev Auth Token Bypass" {
		t.Fatalf("log message = %q", observed.All()[0].Message)
	}
}

// TestAuthenticateIgnoresDevBypassOutsideDevelopment verifies bypass deactivation for non-development environments.
func TestAuthenticateIgnoresDevBypassOutsideDevelopment(t *testing.T) {
	invocations := 0
	service := newServiceForTest(t, "production", "dev-token", "", verifierMock{
		verifyFn: func(ctx context.Context, token string) (*domain.Claims, error) {
			invocations++
			if token != "dev-token" {
				t.Fatalf("token = %q, want %q", token, "dev-token")
			}
			return &domain.Claims{Subject: "user-1", Scope: "contacts:read"}, nil
		},
	})

	err := service.Require(context.Background(), "Bearer dev-token", "contacts:read")
	if err != nil {
		t.Fatalf("Require() error = %v", err)
	}
	if invocations != 1 {
		t.Fatalf("verifier invocations = %d, want %d", invocations, 1)
	}
}

// TestAuthenticateUsesOptionalDevScope verifies optional dev scopes for permission-protected routes.
func TestAuthenticateUsesOptionalDevScope(t *testing.T) {
	service := newServiceForTest(t, "development", "dev-token", "contacts:manage", verifierMock{
		verifyFn: func(ctx context.Context, token string) (*domain.Claims, error) {
			return nil, errors.New("unexpected verifier call")
		},
	})

	err := service.Require(context.Background(), "Bearer dev-token", "contacts:update")
	if err != nil {
		t.Fatalf("Require() error = %v", err)
	}
}

// TestAuthorizeWithoutRequiredPermissions verifies pass-through behavior when no permissions are required.
func TestAuthorizeWithoutRequiredPermissions(t *testing.T) {
	service := newServiceForTest(t, "development", "", "", verifierMock{
		verifyFn: func(ctx context.Context, token string) (*domain.Claims, error) {
			return &domain.Claims{}, nil
		},
	})

	if err := service.Authorize(nil); err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
}

// TestParseBearerToken verifies bearer-token parsing behavior.
func TestParseBearerToken(t *testing.T) {
	token, err := parseBearerToken("Bearer abc")
	if err != nil {
		t.Fatalf("parseBearerToken() error = %v", err)
	}
	if token != "abc" {
		t.Fatalf("token = %q, want %q", token, "abc")
	}

	if _, err := parseBearerToken("Bearer "); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("parseBearerToken() error = %v, want ErrUnauthorized", err)
	}
}

// newServiceForTest creates auth services and fails tests on constructor errors.
func newServiceForTest(t *testing.T, environment string, devToken string, devScope string, verifier verifierMock, loggers ...*zap.Logger) *AuthService {
	t.Helper()

	logger := zap.NewNop()
	if len(loggers) > 0 && loggers[0] != nil {
		logger = loggers[0]
	}

	service, err := NewService(environment, devToken, devScope, verifier, logger)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	return service
}
