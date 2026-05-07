package port

import (
	"context"

	"mannaiah/module/exports/domain"
)

// ListQuery defines export report registry filters.
type ListQuery struct {
	// Type defines optional report type filters.
	Type domain.ReportType
	// Page defines requested page values.
	Page int
	// Limit defines requested page-size values.
	Limit int
}

// Repository defines export report persistence behavior.
type Repository interface {
	// Create persists a generated export report.
	Create(ctx context.Context, report *domain.Report) error
	// GetByID retrieves a generated export report by id.
	GetByID(ctx context.Context, id string) (*domain.Report, error)
	// List returns paginated generated export reports.
	List(ctx context.Context, query ListQuery) ([]domain.Report, int64, error)
}
