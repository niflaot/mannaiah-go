package runtime

import (
	"testing"

	coredatabase "mannaiah/module/core/database"
)

// TestNew verifies module constructor behavior.
func TestNew(t *testing.T) {
	db, err := coredatabase.Open(coredatabase.Config{Driver: "sqlite", DSN: "file::memory:?cache=shared"}, nil)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	module, newErr := New(Config{Enabled: true}, db)
	if newErr != nil {
		t.Fatalf("New() error = %v", newErr)
	}
	if module == nil {
		t.Fatalf("module is nil")
	}
	if module.Recorder() == nil {
		t.Fatalf("module.Recorder() is nil")
	}
}
