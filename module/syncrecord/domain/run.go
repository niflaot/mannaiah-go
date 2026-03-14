package domain

import "time"

// SyncRun defines one synchronization run execution envelope.
type SyncRun struct {
	// ID defines run identifier values.
	ID string `json:"id"`
	// Kind defines synchronization kind values.
	Kind SyncKind `json:"kind"`
	// Trigger defines synchronization trigger values.
	Trigger SyncTrigger `json:"trigger"`
	// Status defines run lifecycle status values.
	Status RunStatus `json:"status"`
	// StartedAt defines run start timestamp values.
	StartedAt time.Time `json:"startedAt"`
	// EndedAt defines optional run end timestamp values.
	EndedAt *time.Time `json:"endedAt,omitempty"`
	// DurationMS defines run duration in milliseconds.
	DurationMS int64 `json:"durationMs"`
	// Processed defines processed item count values.
	Processed int `json:"processed"`
	// Succeeded defines succeeded item count values.
	Succeeded int `json:"succeeded"`
	// Failed defines failed item count values.
	Failed int `json:"failed"`
	// Skipped defines skipped item count values.
	Skipped int `json:"skipped"`
	// ErrorCount defines error row count values.
	ErrorCount int `json:"errorCount"`
	// Metadata defines optional opaque run metadata values.
	Metadata map[string]string `json:"metadata,omitempty"`
	// Errors defines optional run error rows.
	Errors []SyncRunError `json:"errors,omitempty"`
	// CreatedAt defines row creation timestamp values.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt defines row update timestamp values.
	UpdatedAt time.Time `json:"updatedAt"`
}

// RunStats defines aggregate sync-run metrics.
type RunStats struct {
	// WindowStart defines lower-bound timestamp for aggregated counters.
	WindowStart time.Time `json:"windowStart"`
	// TotalRuns defines total runs count in the window.
	TotalRuns int64 `json:"totalRuns"`
	// CompletedRuns defines completed runs count in the window.
	CompletedRuns int64 `json:"completedRuns"`
	// FailedRuns defines failed runs count in the window.
	FailedRuns int64 `json:"failedRuns"`
	// AvgDurationMS defines average run duration in milliseconds.
	AvgDurationMS int64 `json:"avgDurationMs"`
	// LastFailureAt defines optional latest failed-run timestamp in the window.
	LastFailureAt *time.Time `json:"lastFailureAt,omitempty"`
}
