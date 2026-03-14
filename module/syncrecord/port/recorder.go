package port

import (
	"context"
	"time"

	"mannaiah/module/syncrecord/domain"
)

// StartRunInput defines sync run start payload values.
type StartRunInput struct {
	// Kind defines synchronization kind values.
	Kind domain.SyncKind
	// Trigger defines synchronization trigger values.
	Trigger domain.SyncTrigger
	// StartedAt defines optional start timestamp values.
	StartedAt *time.Time
	// Metadata defines optional run metadata values.
	Metadata map[string]string
}

// FinishRunInput defines sync run completion payload values.
type FinishRunInput struct {
	// RunID defines run identifier values.
	RunID string
	// EndedAt defines optional end timestamp values.
	EndedAt *time.Time
	// Processed defines processed item count values.
	Processed int
	// Succeeded defines succeeded item count values.
	Succeeded int
	// Failed defines failed item count values.
	Failed int
	// Skipped defines skipped item count values.
	Skipped int
	// Errors defines optional child error rows for failed runs.
	Errors []domain.SyncRunError
}

// Recorder defines sync-run recording behavior used by cross-module integrations.
type Recorder interface {
	// StartRun starts a running sync run and returns its id.
	StartRun(ctx context.Context, input StartRunInput) (string, error)
	// CompleteRun marks a run as completed.
	CompleteRun(ctx context.Context, input FinishRunInput) error
	// FailRun marks a run as failed and appends error rows.
	FailRun(ctx context.Context, input FinishRunInput) error
}

// NoopRecorder defines no-op recorder behavior for optional wiring.
type NoopRecorder struct{}

// StartRun returns empty run identifiers for no-op recording.
func (NoopRecorder) StartRun(ctx context.Context, input StartRunInput) (string, error) {
	return "", nil
}

// CompleteRun ignores completion payload values.
func (NoopRecorder) CompleteRun(ctx context.Context, input FinishRunInput) error {
	return nil
}

// FailRun ignores failure payload values.
func (NoopRecorder) FailRun(ctx context.Context, input FinishRunInput) error {
	return nil
}
