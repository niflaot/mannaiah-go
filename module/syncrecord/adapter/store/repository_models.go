package store

import "time"

// runModel defines sync run persistence row values.
type runModel struct {
	// ID defines run identifier values.
	ID string `gorm:"column:id;primaryKey"`
	// Kind defines synchronization kind values.
	Kind string `gorm:"column:kind"`
	// Trigger defines synchronization trigger values.
	Trigger string `gorm:"column:sync_trigger"`
	// Status defines run status values.
	Status string `gorm:"column:status"`
	// StartedAt defines run start timestamps.
	StartedAt time.Time `gorm:"column:started_at"`
	// EndedAt defines optional run end timestamps.
	EndedAt *time.Time `gorm:"column:ended_at"`
	// DurationMS defines run duration in milliseconds.
	DurationMS int64 `gorm:"column:duration_ms"`
	// Processed defines processed item count values.
	Processed int `gorm:"column:processed_count"`
	// Succeeded defines succeeded item count values.
	Succeeded int `gorm:"column:succeeded_count"`
	// Failed defines failed item count values.
	Failed int `gorm:"column:failed_count"`
	// Skipped defines skipped item count values.
	Skipped int `gorm:"column:skipped_count"`
	// ErrorCount defines error row count values.
	ErrorCount int `gorm:"column:error_count"`
	// MetadataJSON defines optional metadata json values.
	MetadataJSON string `gorm:"column:metadata_json"`
	// CreatedAt defines row creation timestamps.
	CreatedAt time.Time `gorm:"column:created_at"`
	// UpdatedAt defines row update timestamps.
	UpdatedAt time.Time `gorm:"column:updated_at"`
	// Errors defines run error associations.
	Errors []runErrorModel `gorm:"foreignKey:RunID;references:ID"`
}

// TableName resolves sync run table names.
func (runModel) TableName() string {
	return "sync_runs"
}

// runErrorModel defines sync run error persistence row values.
type runErrorModel struct {
	// ID defines error identifier values.
	ID string `gorm:"column:id;primaryKey"`
	// RunID defines parent run identifier values.
	RunID string `gorm:"column:run_id"`
	// ErrorType defines error category values.
	ErrorType string `gorm:"column:error_type"`
	// ErrorCode defines optional error code values.
	ErrorCode string `gorm:"column:error_code"`
	// Message defines error message values.
	Message string `gorm:"column:message"`
	// CreatedAt defines row creation timestamps.
	CreatedAt time.Time `gorm:"column:created_at"`
}

// TableName resolves sync run error table names.
func (runErrorModel) TableName() string {
	return "sync_run_errors"
}
