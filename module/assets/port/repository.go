package port

import (
	"context"
	"errors"

	"mannaiah/module/assets/domain"
)

var (
	// ErrNotFound is returned when asset records are missing.
	ErrNotFound = errors.New("asset not found")
)

// ListQuery defines list-assets query values.
type ListQuery struct {
	// Page defines page numbers.
	Page int
	// Limit defines page size values.
	Limit int
	// Filters defines optional free-text filters.
	Filters string
}

// PageResult defines paginated list response values.
type PageResult struct {
	// Data defines current page rows.
	Data []domain.Asset
	// Total defines total rows.
	Total int64
	// Page defines current page numbers.
	Page int
	// Limit defines current page size values.
	Limit int
}

// Repository defines asset metadata persistence behavior.
type Repository interface {
	// EnsureSchema ensures storage schema availability.
	EnsureSchema(ctx context.Context) error
	// Create persists asset metadata rows.
	Create(ctx context.Context, asset *domain.Asset) error
	// GetByID loads asset metadata rows by id.
	GetByID(ctx context.Context, id string) (*domain.Asset, error)
	// List paginates asset metadata rows.
	List(ctx context.Context, query ListQuery) (*PageResult, error)
	// UpdateName updates asset custom names.
	UpdateName(ctx context.Context, id string, name string) (*domain.Asset, error)
	// SoftDelete soft-deletes asset metadata rows.
	SoftDelete(ctx context.Context, id string) error
}
