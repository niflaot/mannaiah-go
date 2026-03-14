package port

import "context"

// SyncError defines sync error detail payload values for sync recorder integrations.
type SyncError struct {
	// Type defines high-level error category values.
	Type string
	// Code defines machine-readable error code values.
	Code string
	// Message defines error message values.
	Message string
}

// SyncRecorder defines optional sync run recording behavior.
type SyncRecorder interface {
	// StartRun starts one synchronization run and returns a run identifier.
	StartRun(ctx context.Context, kind string, trigger string) (string, error)
	// CompleteRun marks one synchronization run as completed.
	CompleteRun(ctx context.Context, runID string, processed int, succeeded int, failed int, skipped int) error
	// FailRun marks one synchronization run as failed.
	FailRun(ctx context.Context, runID string, processed int, succeeded int, failed int, skipped int, syncErrors []SyncError) error
}

// NoopSyncRecorder defines no-op sync recorder behavior.
type NoopSyncRecorder struct{}

// StartRun returns empty run identifiers for no-op sync recorder behavior.
func (NoopSyncRecorder) StartRun(ctx context.Context, kind string, trigger string) (string, error) {
	return "", nil
}

// CompleteRun ignores completion payload values for no-op sync recorder behavior.
func (NoopSyncRecorder) CompleteRun(ctx context.Context, runID string, processed int, succeeded int, failed int, skipped int) error {
	return nil
}

// FailRun ignores failure payload values for no-op sync recorder behavior.
func (NoopSyncRecorder) FailRun(ctx context.Context, runID string, processed int, succeeded int, failed int, skipped int, syncErrors []SyncError) error {
	return nil
}
