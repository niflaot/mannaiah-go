package domain

import "strings"

// RunStatus defines execution status of a sync run.
type RunStatus string

const (
	// RunStatusRunning defines in-progress runs.
	RunStatusRunning RunStatus = "running"
	// RunStatusCompleted defines successfully completed runs.
	RunStatusCompleted RunStatus = "completed"
	// RunStatusFailed defines failed runs.
	RunStatusFailed RunStatus = "failed"
)

// IsTerminal reports whether a status is completed or failed.
func (s RunStatus) IsTerminal() bool {
	switch RunStatus(strings.TrimSpace(string(s))) {
	case RunStatusCompleted, RunStatusFailed:
		return true
	default:
		return false
	}
}
