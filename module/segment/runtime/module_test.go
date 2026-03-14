package runtime

import (
	"context"
	"testing"

	analyticsdomain "mannaiah/module/analytics/domain"
	coredatabase "mannaiah/module/core/database"
)

type resolverStub struct{}

func (resolverStub) ResolveContacts(ctx context.Context, filter analyticsdomain.SegmentFilter, page int, limit int) ([]string, error) {
	return []string{}, nil
}

func (resolverStub) CountContacts(ctx context.Context, filter analyticsdomain.SegmentFilter) (int64, error) {
	return 0, nil
}

// TestNew verifies constructor behavior.
func TestNew(t *testing.T) {
	db, err := coredatabase.Open(coredatabase.Config{Driver: "sqlite", DSN: "file::memory:?cache=shared"}, nil)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	module, newErr := New(Config{Enabled: true}, db, resolverStub{})
	if newErr != nil {
		t.Fatalf("New() error = %v", newErr)
	}
	if module == nil || module.Service() == nil {
		t.Fatalf("module or service is nil")
	}
}

// TestNewRejectsNilResolver verifies enabled module constructor validation behavior.
func TestNewRejectsNilResolver(t *testing.T) {
	db, err := coredatabase.Open(coredatabase.Config{Driver: "sqlite", DSN: "file::memory:?cache=shared"}, nil)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	if _, err := New(Config{Enabled: true}, db, nil); err == nil {
		t.Fatalf("New() expected resolver error")
	}
}
