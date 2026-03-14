package port

import (
	"context"
	"time"

	"mannaiah/module/syncrecord/domain"
)

// ListQuery defines repository query filters for listing sync runs.
type ListQuery struct {
	// Kind defines optional sync-kind filters.
	Kind string
	// Trigger defines optional trigger filters.
	Trigger string
	// Status defines optional status filters.
	Status string
	// StartedAfter defines optional lower bound for start timestamps.
	StartedAfter *time.Time
	// StartedBefore defines optional upper bound for start timestamps.
	StartedBefore *time.Time
	// Page defines requested page number.
	Page int
	// Limit defines requested page size.
	Limit int
}

// CompleteInput defines complete-update payload values.
type CompleteInput struct {
	// RunID defines run identifier values.
	RunID string
	// Status defines terminal status values.
	Status domain.RunStatus
	// EndedAt defines completion timestamp values.
	EndedAt time.Time
	// Processed defines processed item count values.
	Processed int
	// Succeeded defines succeeded item count values.
	Succeeded int
	// Failed defines failed item count values.
	Failed int
	// Skipped defines skipped item count values.
	Skipped int
}

// Repository defines persistence behavior required by sync record use-cases.
type Repository interface {
	// CreateRun persists a new running sync run.
	CreateRun(ctx context.Context, run *domain.SyncRun) error
	// CompleteRun updates run status and counters with terminal values.
	CompleteRun(ctx context.Context, input CompleteInput) error
	// AddRunErrors persists child error rows for a run.
	AddRunErrors(ctx context.Context, errors []domain.SyncRunError) error
	// GetRunByID retrieves a run with child errors by id.
	GetRunByID(ctx context.Context, runID string) (*domain.SyncRun, error)
	// ListRuns returns paged run rows and total count for filters.
	ListRuns(ctx context.Context, query ListQuery) ([]domain.SyncRun, int64, error)
	// StatsSince returns aggregate stats from a lower-bound timestamp.
	StatsSince(ctx context.Context, since time.Time) (*domain.RunStats, error)
	// CleanupBefore deletes runs older than cutoff and returns deleted rows count.
	CleanupBefore(ctx context.Context, cutoff time.Time) (int64, error)
}
