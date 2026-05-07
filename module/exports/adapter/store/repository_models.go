package store

import "time"

// reportModel defines generated export report registry rows.
type reportModel struct {
	// ID defines report identifier values.
	ID string `gorm:"column:id;primaryKey"`
	// ReportType defines exported resource type values.
	ReportType string `gorm:"column:report_type"`
	// Status defines generated report status values.
	Status string `gorm:"column:status"`
	// Stamp defines deterministic timestamp labels used in filenames.
	Stamp string `gorm:"column:stamp"`
	// FileName defines generated CSV file names.
	FileName string `gorm:"column:file_name"`
	// StorageKey defines MinIO/S3 object storage keys.
	StorageKey string `gorm:"column:storage_key"`
	// SHA256Hash defines generated report content hashes.
	SHA256Hash string `gorm:"column:sha256_hash"`
	// ContentType defines generated object content types.
	ContentType string `gorm:"column:content_type"`
	// RowCount defines exported data row counts.
	RowCount int `gorm:"column:row_count"`
	// ByteSize defines generated CSV byte size values.
	ByteSize int64 `gorm:"column:byte_size"`
	// GeneratedAt defines report generation timestamps.
	GeneratedAt time.Time `gorm:"column:generated_at"`
	// CreatedAt defines registry creation timestamps.
	CreatedAt time.Time `gorm:"column:created_at"`
	// UpdatedAt defines registry update timestamps.
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

// TableName resolves generated export report registry table names.
func (reportModel) TableName() string {
	return "export_reports"
}
