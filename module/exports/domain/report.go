package domain

import (
	"errors"
	"strings"
	"time"
)

const (
	// ReportTypeContacts identifies contact export reports.
	ReportTypeContacts ReportType = "contacts"
	// ReportTypeOrders identifies order export reports.
	ReportTypeOrders ReportType = "orders"
	// ReportStatusCompleted identifies successfully generated reports.
	ReportStatusCompleted ReportStatus = "completed"
)

var (
	// ErrInvalidReportType is returned when report types are unsupported.
	ErrInvalidReportType = errors.New("export report type is invalid")
	// ErrInvalidReportID is returned when report identifiers are empty.
	ErrInvalidReportID = errors.New("export report id is required")
	// ErrReportNotFound is returned when an export report cannot be found.
	ErrReportNotFound = errors.New("export report not found")
)

// ReportType identifies export report categories.
type ReportType string

// IsValid reports whether a report type is supported.
func (t ReportType) IsValid() bool {
	switch t {
	case ReportTypeContacts, ReportTypeOrders:
		return true
	default:
		return false
	}
}

// Normalize trims report type values.
func (t ReportType) Normalize() ReportType {
	return ReportType(strings.ToLower(strings.TrimSpace(string(t))))
}

// ReportStatus identifies export report lifecycle status values.
type ReportStatus string

// Report defines generated export report registry values.
type Report struct {
	// ID defines report identifier values.
	ID string `json:"id"`
	// Type defines exported resource type values.
	Type ReportType `json:"type"`
	// Status defines generated report status values.
	Status ReportStatus `json:"status"`
	// Stamp defines deterministic timestamp labels used in filenames.
	Stamp string `json:"stamp"`
	// FileName defines generated CSV file names.
	FileName string `json:"fileName"`
	// StorageKey defines MinIO/S3 object storage keys.
	StorageKey string `json:"storageKey"`
	// SHA256 defines generated report content hashes.
	SHA256 string `json:"sha256"`
	// ContentType defines generated object content types.
	ContentType string `json:"contentType"`
	// RowCount defines exported data row counts.
	RowCount int `json:"rowCount"`
	// ByteSize defines generated CSV byte size values.
	ByteSize int64 `json:"byteSize"`
	// GeneratedAt defines report generation timestamps.
	GeneratedAt time.Time `json:"generatedAt"`
	// CreatedAt defines registry creation timestamps.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt defines registry update timestamps.
	UpdatedAt time.Time `json:"updatedAt"`
}

// Normalize trims report fields and applies default status values.
func (r *Report) Normalize() {
	if r == nil {
		return
	}
	r.ID = strings.TrimSpace(r.ID)
	r.Type = r.Type.Normalize()
	r.Status = ReportStatus(strings.TrimSpace(string(r.Status)))
	if r.Status == "" {
		r.Status = ReportStatusCompleted
	}
	r.Stamp = strings.TrimSpace(r.Stamp)
	r.FileName = strings.TrimSpace(r.FileName)
	r.StorageKey = strings.TrimSpace(r.StorageKey)
	r.SHA256 = strings.TrimSpace(r.SHA256)
	r.ContentType = strings.TrimSpace(r.ContentType)
}

// Validate verifies report registry values.
func (r Report) Validate() error {
	normalizedType := r.Type.Normalize()
	if !normalizedType.IsValid() {
		return ErrInvalidReportType
	}
	if strings.TrimSpace(r.ID) == "" {
		return ErrInvalidReportID
	}
	return nil
}
