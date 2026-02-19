package port

import (
	"context"
	"errors"
	"time"

	syncdomain "mannaiah/module/falabella/domain/sync"
)

var (
	// ErrSyncEntryNotFound is returned when a sync status entry is not found.
	ErrSyncEntryNotFound = errors.New("falabella sync entry not found")
	// ErrDuplicateFeedID is returned when a feed ID already exists.
	ErrDuplicateFeedID = errors.New("falabella feed id already exists")
)

// SyncStatusRepository defines persistence behavior for Falabella sync status entries.
type SyncStatusRepository interface {
	// EnsureSchema migrates sync status persistence schema.
	EnsureSchema(ctx context.Context) error
	// Create persists a new sync status entry.
	Create(ctx context.Context, entry *syncdomain.SyncEntry) error
	// GetByFeedID retrieves a sync status entry by Falabella feed identifier.
	GetByFeedID(ctx context.Context, feedID string) (*syncdomain.SyncEntry, error)
	// GetByProductID retrieves sync status entries by source product identifier.
	GetByProductID(ctx context.Context, productID string) ([]syncdomain.SyncEntry, error)
	// ListPending retrieves unresolved sync status entries ordered by submission time.
	ListPending(ctx context.Context, limit int) ([]syncdomain.SyncEntry, error)
	// UpdateStatus updates the status and resolution timestamp of a sync status entry.
	UpdateStatus(ctx context.Context, feedID string, status syncdomain.SyncStatus, resolvedAt *time.Time) error
}
