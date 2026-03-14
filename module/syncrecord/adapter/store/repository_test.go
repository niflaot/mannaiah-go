package store

import (
	"context"
	"testing"
	"time"

	"gorm.io/gorm"
	coredatabase "mannaiah/module/core/database"
	"mannaiah/module/syncrecord/domain"
	"mannaiah/module/syncrecord/port"
)

// TestRepositoryLifecycle verifies create/list/complete flow behavior.
func TestRepositoryLifecycle(t *testing.T) {
	db, err := coredatabase.Open(coredatabase.Config{Driver: "sqlite", DSN: "file::memory:?cache=shared"}, nil)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	applySchemaForTest(t, db)

	repo, err := NewRepository(db)
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}

	run := &domain.SyncRun{
		ID:        "run-1",
		Kind:      domain.KindWooCommerceContacts,
		Trigger:   domain.TriggerManual,
		Status:    domain.RunStatusRunning,
		StartedAt: time.Now().UTC(),
	}
	if createErr := repo.CreateRun(context.Background(), run); createErr != nil {
		t.Fatalf("CreateRun() error = %v", createErr)
	}

	if addErr := repo.AddRunErrors(context.Background(), []domain.SyncRunError{{
		ID:        "error-1",
		RunID:     "run-1",
		ErrorType: "dependency",
		Message:   "connection timeout",
		CreatedAt: time.Now().UTC(),
	}}); addErr != nil {
		t.Fatalf("AddRunErrors() error = %v", addErr)
	}

	if completeErr := repo.CompleteRun(context.Background(), port.CompleteInput{
		RunID:     "run-1",
		Status:    domain.RunStatusFailed,
		EndedAt:   time.Now().UTC(),
		Processed: 10,
		Succeeded: 8,
		Failed:    2,
		Skipped:   0,
	}); completeErr != nil {
		t.Fatalf("CompleteRun() error = %v", completeErr)
	}

	loaded, loadErr := repo.GetRunByID(context.Background(), "run-1")
	if loadErr != nil {
		t.Fatalf("GetRunByID() error = %v", loadErr)
	}
	if loaded.Status != domain.RunStatusFailed {
		t.Fatalf("loaded.Status = %q, want failed", loaded.Status)
	}
	if loaded.ErrorCount != 1 {
		t.Fatalf("loaded.ErrorCount = %d, want 1", loaded.ErrorCount)
	}

	rows, total, listErr := repo.ListRuns(context.Background(), port.ListQuery{Page: 1, Limit: 10})
	if listErr != nil {
		t.Fatalf("ListRuns() error = %v", listErr)
	}
	if total != 1 || len(rows) != 1 {
		t.Fatalf("ListRuns() total=%d len=%d, want 1", total, len(rows))
	}
}

// applySchemaForTest creates storage tables for sqlite repository tests.
func applySchemaForTest(t *testing.T, db *gorm.DB) {
	t.Helper()

	queries := []string{
		`CREATE TABLE sync_runs (
			id TEXT PRIMARY KEY,
			kind TEXT NOT NULL,
			trigger TEXT NOT NULL,
			status TEXT NOT NULL,
			started_at DATETIME NOT NULL,
			ended_at DATETIME,
			duration_ms INTEGER NOT NULL DEFAULT 0,
			processed_count INTEGER NOT NULL DEFAULT 0,
			succeeded_count INTEGER NOT NULL DEFAULT 0,
			failed_count INTEGER NOT NULL DEFAULT 0,
			skipped_count INTEGER NOT NULL DEFAULT 0,
			error_count INTEGER NOT NULL DEFAULT 0,
			metadata_json TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE sync_run_errors (
			id TEXT PRIMARY KEY,
			run_id TEXT NOT NULL,
			error_type TEXT NOT NULL,
			error_code TEXT,
			message TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(run_id) REFERENCES sync_runs(id) ON DELETE CASCADE
		);`,
	}

	for _, query := range queries {
		if err := db.Exec(query).Error; err != nil {
			t.Fatalf("Exec(%q) error = %v", query, err)
		}
	}
}
