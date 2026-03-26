package store

import (
	"context"
	"testing"

	coredb "mannaiah/module/core/database"
	coredbmigration "mannaiah/module/core/database/migration"
	"mannaiah/module/email/domain"
)

// TestRepositoryListByEmail verifies recipient email filtering and deterministic ordering behavior.
func TestRepositoryListByEmail(t *testing.T) {
	repository := newRepositoryForTest(t)
	seed := []*domain.Delivery{
		{ID: "d-1", Email: "user@example.com", Subject: "Subject 1", HTMLBody: "<p>One</p>", TextBody: "One", IdempotencyKey: "idem-1", Provider: "ses", Status: domain.StatusSubmitted},
		{ID: "d-2", Email: "other@example.com", Subject: "Subject 2", HTMLBody: "<p>Two</p>", TextBody: "Two", IdempotencyKey: "idem-2", Provider: "ses", Status: domain.StatusSubmitted},
		{ID: "d-3", Email: "USER@example.com", Subject: "Subject 3", HTMLBody: "<p>Three</p>", TextBody: "Three", IdempotencyKey: "idem-3", Provider: "ses", Status: domain.StatusSubmitted},
	}
	for _, delivery := range seed {
		if err := repository.CreateDelivery(context.Background(), delivery); err != nil {
			t.Fatalf("CreateDelivery(%s) error = %v", delivery.ID, err)
		}
	}

	rows, err := repository.ListByEmail(context.Background(), " user@example.com ")
	if err != nil {
		t.Fatalf("ListByEmail() error = %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}
	if rows[0].ID != "d-3" || rows[1].ID != "d-1" {
		t.Fatalf("rows order = [%s, %s], want [d-3, d-1]", rows[0].ID, rows[1].ID)
	}
}

// TestRepositoryListByEmailNoMatches verifies missing recipient email behavior.
func TestRepositoryListByEmailNoMatches(t *testing.T) {
	repository := newRepositoryForTest(t)
	rows, err := repository.ListByEmail(context.Background(), "missing@example.com")
	if err != nil {
		t.Fatalf("ListByEmail() error = %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("len(rows) = %d, want 0", len(rows))
	}
}

// newRepositoryForTest creates a migration-ready email repository for tests.
func newRepositoryForTest(t *testing.T) *Repository {
	t.Helper()

	db, err := coredb.Open(coredb.Config{Driver: "sqlite", DSN: "file::memory:?cache=shared", MaxOpenConns: 1}, nil)
	if err != nil {
		t.Fatalf("coredb.Open() error = %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	if err := coredbmigration.Apply(context.Background(), db, coredbmigration.Config{Enabled: true, Driver: "sqlite", Table: "schema_migrations"}, nil); err != nil {
		t.Fatalf("coredbmigration.Apply() error = %v", err)
	}

	repository, err := NewRepository(db)
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}

	return repository
}
