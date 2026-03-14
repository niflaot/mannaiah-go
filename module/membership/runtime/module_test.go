package runtime

import (
	"context"
	"testing"

	coredatabase "mannaiah/module/core/database"
	"mannaiah/module/membership/port"
)

// contactLookupMock defines contact lookup behavior for runtime tests.
type contactLookupMock struct{}

// FindByEmail resolves one contact by normalized email values.
func (contactLookupMock) FindByEmail(ctx context.Context, email string) (*port.ContactSnapshot, error) {
	return &port.ContactSnapshot{ID: "c-1", Email: email}, nil
}

// ListByMetadata resolves contacts by metadata key/value filters.
func (contactLookupMock) ListByMetadata(ctx context.Context, metadataKey string, metadataValue string, page int, limit int) ([]port.ContactSnapshot, int64, error) {
	return nil, 0, nil
}

// TestNew verifies constructor behavior.
func TestNew(t *testing.T) {
	db, err := coredatabase.Open(coredatabase.Config{Driver: "sqlite", DSN: "file::memory:?cache=shared"}, nil)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	module, newErr := New(Config{Enabled: true}, db, contactLookupMock{})
	if newErr != nil {
		t.Fatalf("New() error = %v", newErr)
	}
	if module == nil || module.Service() == nil {
		t.Fatalf("module or service is nil")
	}
}
