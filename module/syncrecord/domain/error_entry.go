package domain

import "time"

// SyncRunError represents one error captured during a sync run.
type SyncRunError struct {
	// ID defines a stable error row identifier.
	ID string `json:"id"`
	// RunID defines parent run identifier values.
	RunID string `json:"runId"`
	// ErrorType defines category values such as validation/dependency/timeout.
	ErrorType string `json:"errorType"`
	// ErrorCode defines optional machine-readable codes.
	ErrorCode string `json:"errorCode,omitempty"`
	// Message defines error detail values.
	Message string `json:"message"`
	// CreatedAt defines row creation timestamp values.
	CreatedAt time.Time `json:"createdAt"`
}
