package runtime

import (
	"testing"

	coredatabase "mannaiah/module/core/database"
)

// TestNew verifies shipping runtime constructor behavior.
func TestNew(t *testing.T) {
	db, err := coredatabase.Open(coredatabase.Config{Driver: "sqlite", DSN: "file::memory:?cache=shared"}, nil)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	module, newErr := New(Config{Enabled: true}, db)
	if newErr != nil {
		t.Fatalf("New() error = %v", newErr)
	}
	if module == nil || module.MarkService() == nil || module.DispatchService() == nil {
		t.Fatalf("module services are nil")
	}
}

// TestResolveTCCAccessToken verifies sandbox/production token selection behavior.
func TestResolveTCCAccessToken(t *testing.T) {
	if token := resolveTCCAccessToken(TCCConfig{
		Sandbox:               true,
		SandboxAccessToken:    "sandbox-token",
		ProductionAccessToken: "production-token",
	}); token != "sandbox-token" {
		t.Fatalf("sandbox token = %q", token)
	}
	if token := resolveTCCAccessToken(TCCConfig{
		Sandbox:               false,
		SandboxAccessToken:    "sandbox-token",
		ProductionAccessToken: "production-token",
	}); token != "production-token" {
		t.Fatalf("production token = %q", token)
	}
}
