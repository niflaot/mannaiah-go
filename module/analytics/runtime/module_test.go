package runtime

import (
	"testing"

	coredatabase "mannaiah/module/core/database"
)

// TestNew verifies constructor behavior.
func TestNew(t *testing.T) {
	db, err := coredatabase.Open(coredatabase.Config{Driver: "sqlite", DSN: "file::memory:?cache=shared"}, nil)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	module, newErr := New(Config{Enabled: false}, db, nil)
	if newErr != nil {
		t.Fatalf("New() error = %v", newErr)
	}
	if module == nil || module.QueryService() == nil {
		t.Fatalf("module or resolver is nil")
	}
}

// TestNewEnabledRequiresDSN verifies enabled analytics constructor validation behavior.
func TestNewEnabledRequiresDSN(t *testing.T) {
	db, err := coredatabase.Open(coredatabase.Config{Driver: "sqlite", DSN: "file::memory:?cache=shared"}, nil)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	if _, err := New(Config{Enabled: true}, db, nil); err == nil {
		t.Fatalf("New(enabled without dsn) expected error")
	}
}
