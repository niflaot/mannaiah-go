package store

import (
	"context"
	"errors"
	"testing"
	"time"

	syncdomain "mannaiah/module/falabella/domain/sync"
	"mannaiah/module/falabella/port"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newTestDB creates in-memory SQLite databases for sync status repository tests.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	return db
}

// TestNewRepositoryNilDB verifies constructor nil-DB validation behavior.
func TestNewRepositoryNilDB(t *testing.T) {
	_, err := NewRepository(nil)
	if !errors.Is(err, ErrNilDB) {
		t.Fatalf("NewRepository(nil) error = %v, want %v", err, ErrNilDB)
	}
}

// TestEnsureSchema verifies schema migration behavior.
func TestEnsureSchema(t *testing.T) {
	db := newTestDB(t)
	repo, err := NewRepository(db)
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}

	if err := repo.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema() error = %v", err)
	}
}

// TestEnsureSchemaMigrationFromLegacy verifies legacy schema (with id column) is dropped and recreated.
func TestEnsureSchemaMigrationFromLegacy(t *testing.T) {
	db := newTestDB(t)

	// Create legacy table with id column.
	if err := db.Exec(`CREATE TABLE falabella_sync_status (
		id VARCHAR(64) PRIMARY KEY,
		feed_id VARCHAR(191) NOT NULL,
		product_id VARCHAR(128) NOT NULL,
		sku VARCHAR(128) NOT NULL,
		action VARCHAR(16) NOT NULL,
		status VARCHAR(16) NOT NULL,
		synced_at DATETIME NOT NULL,
		resolved_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("create legacy table: %v", err)
	}

	repo, _ := NewRepository(db)
	if err := repo.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema() error = %v", err)
	}

	// Verify we can create an entry with feed_id as PK (no id column).
	entry := &syncdomain.SyncEntry{
		ProductID: "prod-1", SKU: "SKU-001", FeedID: "feed-legacy",
		Action:    syncdomain.SyncActionCreate,
		Status:    syncdomain.SyncStatusPending,
		SyncedAt:  time.Now().UTC(),
	}
	if err := repo.Create(context.Background(), entry); err != nil {
		t.Fatalf("Create() after legacy migration error = %v", err)
	}

	retrieved, err := repo.GetByFeedID(context.Background(), "feed-legacy")
	if err != nil {
		t.Fatalf("GetByFeedID() error = %v", err)
	}
	if retrieved.FeedID != "feed-legacy" {
		t.Fatalf("FeedID = %q, want %q", retrieved.FeedID, "feed-legacy")
	}
}

// TestCreateAndGetByFeedID verifies create and feed-ID lookup behavior.
func TestCreateAndGetByFeedID(t *testing.T) {
	db := newTestDB(t)
	repo, _ := NewRepository(db)
	_ = repo.EnsureSchema(context.Background())

	entry := &syncdomain.SyncEntry{
		ProductID: "prod-1",
		SKU:       "SKU-001",
		FeedID:    "feed-abc",
		Action:    syncdomain.SyncActionCreate,
		Status:    syncdomain.SyncStatusPending,
		SyncedAt:  time.Now().UTC().Truncate(time.Second),
	}

	if err := repo.Create(context.Background(), entry); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	retrieved, err := repo.GetByFeedID(context.Background(), "feed-abc")
	if err != nil {
		t.Fatalf("GetByFeedID() error = %v", err)
	}
	if retrieved.ProductID != "prod-1" {
		t.Fatalf("ProductID = %q, want %q", retrieved.ProductID, "prod-1")
	}
	if retrieved.SKU != "SKU-001" {
		t.Fatalf("SKU = %q, want %q", retrieved.SKU, "SKU-001")
	}
	if retrieved.FeedID != "feed-abc" {
		t.Fatalf("FeedID = %q, want %q", retrieved.FeedID, "feed-abc")
	}
	if retrieved.Action != syncdomain.SyncActionCreate {
		t.Fatalf("Action = %q, want %q", retrieved.Action, syncdomain.SyncActionCreate)
	}
	if retrieved.Status != syncdomain.SyncStatusPending {
		t.Fatalf("Status = %q, want %q", retrieved.Status, syncdomain.SyncStatusPending)
	}
}

// TestGetByFeedIDNotFound verifies not-found error behavior.
func TestGetByFeedIDNotFound(t *testing.T) {
	db := newTestDB(t)
	repo, _ := NewRepository(db)
	_ = repo.EnsureSchema(context.Background())

	_, err := repo.GetByFeedID(context.Background(), "nonexistent")
	if !errors.Is(err, port.ErrSyncEntryNotFound) {
		t.Fatalf("GetByFeedID() error = %v, want %v", err, port.ErrSyncEntryNotFound)
	}
}

// TestGetByProductID verifies product-ID lookup behavior.
func TestGetByProductID(t *testing.T) {
	db := newTestDB(t)
	repo, _ := NewRepository(db)
	_ = repo.EnsureSchema(context.Background())

	now := time.Now().UTC().Truncate(time.Second)
	_ = repo.Create(context.Background(), &syncdomain.SyncEntry{
		ProductID: "prod-1", SKU: "SKU-001", FeedID: "feed-1",
		Action: syncdomain.SyncActionCreate, Status: syncdomain.SyncStatusPending, SyncedAt: now,
	})
	_ = repo.Create(context.Background(), &syncdomain.SyncEntry{
		ProductID: "prod-1", SKU: "SKU-001", FeedID: "feed-2",
		Action: syncdomain.SyncActionUpdate, Status: syncdomain.SyncStatusFinished, SyncedAt: now.Add(time.Minute),
	})
	_ = repo.Create(context.Background(), &syncdomain.SyncEntry{
		ProductID: "prod-2", SKU: "SKU-002", FeedID: "feed-3",
		Action: syncdomain.SyncActionCreate, Status: syncdomain.SyncStatusPending, SyncedAt: now,
	})

	entries, err := repo.GetByProductID(context.Background(), "prod-1")
	if err != nil {
		t.Fatalf("GetByProductID() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want %d", len(entries), 2)
	}
}

// TestUpdateStatus verifies status update behavior.
func TestUpdateStatus(t *testing.T) {
	db := newTestDB(t)
	repo, _ := NewRepository(db)
	_ = repo.EnsureSchema(context.Background())

	now := time.Now().UTC().Truncate(time.Second)
	_ = repo.Create(context.Background(), &syncdomain.SyncEntry{
		ProductID: "prod-1", SKU: "SKU-001", FeedID: "feed-update",
		Action: syncdomain.SyncActionCreate, Status: syncdomain.SyncStatusPending, SyncedAt: now,
	})

	resolvedAt := now.Add(5 * time.Minute)
	if err := repo.UpdateStatus(context.Background(), "feed-update", syncdomain.SyncStatusFinished, &resolvedAt); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}

	retrieved, _ := repo.GetByFeedID(context.Background(), "feed-update")
	if retrieved.Status != syncdomain.SyncStatusFinished {
		t.Fatalf("Status = %q, want %q", retrieved.Status, syncdomain.SyncStatusFinished)
	}
	if retrieved.ResolvedAt == nil {
		t.Fatalf("ResolvedAt should not be nil")
	}
}

// TestUpdateStatusNotFound verifies not-found error on update behavior.
func TestUpdateStatusNotFound(t *testing.T) {
	db := newTestDB(t)
	repo, _ := NewRepository(db)
	_ = repo.EnsureSchema(context.Background())

	now := time.Now().UTC()
	err := repo.UpdateStatus(context.Background(), "nonexistent", syncdomain.SyncStatusFinished, &now)
	if !errors.Is(err, port.ErrSyncEntryNotFound) {
		t.Fatalf("UpdateStatus() error = %v, want %v", err, port.ErrSyncEntryNotFound)
	}
}

// TestCreateDuplicateFeedID verifies duplicate feed-ID error behavior.
func TestCreateDuplicateFeedID(t *testing.T) {
	db := newTestDB(t)
	repo, _ := NewRepository(db)
	_ = repo.EnsureSchema(context.Background())

	now := time.Now().UTC().Truncate(time.Second)
	entry := &syncdomain.SyncEntry{
		ProductID: "prod-1", SKU: "SKU-001", FeedID: "feed-dup",
		Action: syncdomain.SyncActionCreate, Status: syncdomain.SyncStatusPending, SyncedAt: now,
	}
	_ = repo.Create(context.Background(), entry)

	duplicate := &syncdomain.SyncEntry{
		ProductID: "prod-1", SKU: "SKU-001", FeedID: "feed-dup",
		Action: syncdomain.SyncActionCreate, Status: syncdomain.SyncStatusPending, SyncedAt: now,
	}
	err := repo.Create(context.Background(), duplicate)
	if err == nil {
		t.Fatalf("Create() should fail with duplicate feed ID")
	}
}

// TestCreateDuplicateFeedIDRollsBackExecution verifies transactional rollback for execution rows when entry insert fails.
func TestCreateDuplicateFeedIDRollsBackExecution(t *testing.T) {
	db := newTestDB(t)
	repo, _ := NewRepository(db)
	_ = repo.EnsureSchema(context.Background())

	now := time.Now().UTC().Truncate(time.Second)
	first := &syncdomain.SyncEntry{
		ExecutionID: "exec-1",
		ProductID:   "prod-1",
		SKU:         "SKU-001",
		FeedID:      "feed-dup-rb",
		Action:      syncdomain.SyncActionCreate,
		Status:      syncdomain.SyncStatusPending,
		SyncedAt:    now,
	}
	if err := repo.Create(context.Background(), first); err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}

	duplicate := &syncdomain.SyncEntry{
		ExecutionID: "exec-2",
		ProductID:   "prod-2",
		SKU:         "SKU-002",
		FeedID:      "feed-dup-rb",
		Action:      syncdomain.SyncActionCreate,
		Status:      syncdomain.SyncStatusPending,
		SyncedAt:    now,
	}
	err := repo.Create(context.Background(), duplicate)
	if err == nil {
		t.Fatalf("Create(duplicate) should fail")
	}

	_, getErr := repo.GetExecutionByID(context.Background(), "exec-2")
	if !errors.Is(getErr, port.ErrSyncExecutionNotFound) {
		t.Fatalf("GetExecutionByID(exec-2) error = %v, want %v", getErr, port.ErrSyncExecutionNotFound)
	}
}

// TestListPending verifies pending entry retrieval behavior.
func TestListPending(t *testing.T) {
	db := newTestDB(t)
	repo, _ := NewRepository(db)
	_ = repo.EnsureSchema(context.Background())

	now := time.Now().UTC().Truncate(time.Second)
	_ = repo.Create(context.Background(), &syncdomain.SyncEntry{
		ProductID: "prod-1", SKU: "SKU-001", FeedID: "feed-pending-1",
		Action: syncdomain.SyncActionCreate, Status: syncdomain.SyncStatusPending, SyncedAt: now,
	})
	_ = repo.Create(context.Background(), &syncdomain.SyncEntry{
		ProductID: "prod-2", SKU: "SKU-002", FeedID: "feed-pending-2",
		Action: syncdomain.SyncActionCreate, Status: syncdomain.SyncStatusPending, SyncedAt: now.Add(time.Minute),
	})
	_ = repo.Create(context.Background(), &syncdomain.SyncEntry{
		ProductID: "prod-3", SKU: "SKU-003", FeedID: "feed-finished",
		Action: syncdomain.SyncActionCreate, Status: syncdomain.SyncStatusFinished, SyncedAt: now.Add(2 * time.Minute),
	})

	entries, err := repo.ListPending(context.Background(), 50)
	if err != nil {
		t.Fatalf("ListPending() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want %d", len(entries), 2)
	}
	if entries[0].FeedID != "feed-pending-1" {
		t.Fatalf("entries[0].FeedID = %q, want %q (should be ordered by synced_at ASC)", entries[0].FeedID, "feed-pending-1")
	}
	if entries[1].FeedID != "feed-pending-2" {
		t.Fatalf("entries[1].FeedID = %q, want %q", entries[1].FeedID, "feed-pending-2")
	}
}

// TestListPendingLimit verifies limit enforcement behavior.
func TestListPendingLimit(t *testing.T) {
	db := newTestDB(t)
	repo, _ := NewRepository(db)
	_ = repo.EnsureSchema(context.Background())

	now := time.Now().UTC().Truncate(time.Second)
	for i := 0; i < 5; i++ {
		_ = repo.Create(context.Background(), &syncdomain.SyncEntry{
			ProductID: "prod-1", SKU: "SKU-001", FeedID: "feed-limit-" + time.Duration(i).String(),
			Action: syncdomain.SyncActionCreate, Status: syncdomain.SyncStatusPending, SyncedAt: now.Add(time.Duration(i) * time.Minute),
		})
	}

	entries, err := repo.ListPending(context.Background(), 2)
	if err != nil {
		t.Fatalf("ListPending() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want %d", len(entries), 2)
	}
}

// TestListPendingDefaultLimit verifies zero limit defaults behavior.
func TestListPendingDefaultLimit(t *testing.T) {
	db := newTestDB(t)
	repo, _ := NewRepository(db)
	_ = repo.EnsureSchema(context.Background())

	entries, err := repo.ListPending(context.Background(), 0)
	if err != nil {
		t.Fatalf("ListPending() error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("len(entries) = %d, want %d", len(entries), 0)
	}
}

// TestListPendingEmpty verifies empty result behavior.
func TestListPendingEmpty(t *testing.T) {
	db := newTestDB(t)
	repo, _ := NewRepository(db)
	_ = repo.EnsureSchema(context.Background())

	entries, err := repo.ListPending(context.Background(), 50)
	if err != nil {
		t.Fatalf("ListPending() error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("len(entries) = %d, want %d", len(entries), 0)
	}
}