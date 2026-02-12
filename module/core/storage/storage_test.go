package storage

import (
	errorspkg "errors"
	"testing"
)

// TestFacadeConstructors verifies storage facade constructor behavior.
func TestFacadeConstructors(t *testing.T) {
	disabled := Disabled(errorspkg.New("disabled"))
	if disabled == nil {
		t.Fatalf("Disabled() should not return nil")
	}
	if disabled.AvailabilityError() == nil {
		t.Fatalf("expected availability error from disabled store")
	}

	store := NewS3(Config{Enabled: false}, nil)
	if store == nil {
		t.Fatalf("NewS3() should not return nil")
	}
}
