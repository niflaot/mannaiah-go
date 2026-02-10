package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
	"mannaiah/module/auth/application"
)

// TestResolveEnvironment verifies auth environment resolution from core environment.
func TestResolveEnvironment(t *testing.T) {
	if resolved := resolveEnvironment("development"); resolved != "development" {
		t.Fatalf("resolveEnvironment() = %q, want %q", resolved, "development")
	}
	if resolved := resolveEnvironment("production"); resolved != "production" {
		t.Fatalf("resolveEnvironment() = %q, want %q", resolved, "production")
	}
	if resolved := resolveEnvironment(""); resolved != "development" {
		t.Fatalf("resolveEnvironment() = %q, want %q", resolved, "development")
	}
}

// TestBuildJWKSURL verifies JWKS URL derivation behavior.
func TestBuildJWKSURL(t *testing.T) {
	if value := buildJWKSURL("https://issuer.example"); value != "https://issuer.example/jwks" {
		t.Fatalf("buildJWKSURL() = %q", value)
	}
	if value := buildJWKSURL("https://issuer.example/"); value != "https://issuer.example/jwks" {
		t.Fatalf("buildJWKSURL() = %q", value)
	}
	if value := buildJWKSURL(" "); value != "" {
		t.Fatalf("buildJWKSURL() = %q, want empty", value)
	}
}

// TestResolvePositiveInt verifies positive-int fallback behavior.
func TestResolvePositiveInt(t *testing.T) {
	if value := resolvePositiveInt(1, 2); value != 1 {
		t.Fatalf("resolvePositiveInt() = %d, want %d", value, 1)
	}
	if value := resolvePositiveInt(0, 2); value != 2 {
		t.Fatalf("resolvePositiveInt() = %d, want %d", value, 2)
	}
}

// TestModuleNilRequire verifies nil-module authorization behavior.
func TestModuleNilRequire(t *testing.T) {
	var module *Module
	err := module.Require(context.Background(), "Bearer token", "contacts:read")
	if !errors.Is(err, application.ErrUnauthorized) {
		t.Fatalf("Require() error = %v, want ErrUnauthorized", err)
	}
}

// TestModuleErrorClassifiers verifies auth error classification behavior.
func TestModuleErrorClassifiers(t *testing.T) {
	module := &Module{}
	if !module.IsUnauthorized(application.ErrUnauthorized) {
		t.Fatalf("expected unauthorized classifier")
	}
	if !module.IsForbidden(application.ErrForbidden) {
		t.Fatalf("expected forbidden classifier")
	}
}

// TestNewRejectsInvalidConfig verifies module constructor validation behavior.
func TestNewRejectsInvalidConfig(t *testing.T) {
	_, err := New(Config{}, "development", zap.NewNop())
	if err == nil {
		t.Fatalf("expected constructor error for invalid config")
	}
}

// TestNewWithDevBypass verifies development bypass behavior on initialized module instances.
func TestNewWithDevBypass(t *testing.T) {
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"keys":[]}`))
	}))
	defer jwksServer.Close()

	module, err := New(Config{
		Issuer:                 jwksServer.URL,
		Audience:               "https://api.mannaiah.test",
		DevAuthToken:           "dev-token",
		DevAuthScope:           "contacts:manage",
		JWKSRateLimitPerMinute: 5,
		JWKSCacheTTLMS:         300000,
		JWKSHTTPTimeoutMS:      5000,
	}, "development", zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	requireErr := module.Require(context.Background(), "Bearer dev-token", "contacts:update")
	if requireErr != nil {
		t.Fatalf("Require() error = %v", requireErr)
	}
}

// TestNewWithDevBypassMissingScope verifies forbidden behavior when bypass scope is not configured.
func TestNewWithDevBypassMissingScope(t *testing.T) {
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"keys":[]}`))
	}))
	defer jwksServer.Close()

	module, err := New(Config{
		Issuer:                 jwksServer.URL,
		Audience:               "https://api.mannaiah.test",
		DevAuthToken:           "dev-token",
		JWKSRateLimitPerMinute: 5,
		JWKSCacheTTLMS:         300000,
		JWKSHTTPTimeoutMS:      5000,
	}, "development", zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	requireErr := module.Require(context.Background(), "Bearer dev-token", "contacts:update")
	if !module.IsForbidden(requireErr) {
		t.Fatalf("Require() error = %v, want forbidden", requireErr)
	}
}
