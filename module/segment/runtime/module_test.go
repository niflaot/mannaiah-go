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

	module, newErr := New(Config{Enabled: true}, db, nil)
	if newErr != nil {
		t.Fatalf("New() error = %v", newErr)
	}
	if module == nil || module.Service() == nil {
		t.Fatalf("module or service is nil")
	}
}
