package variation

import (
	"context"
	"errors"

	variationdomain "mannaiah/module/products/domain/variation"
)

var (
	// ErrNotFound is returned when variation records are missing.
	ErrNotFound = errors.New("variation not found")
)

// Repository defines variation persistence contracts.
type Repository interface {
	// EnsureSchema ensures schema availability.
	EnsureSchema(ctx context.Context) error
	// Create persists variations.
	Create(ctx context.Context, variation *variationdomain.Variation) error
	// GetByID retrieves variations by ID.
	GetByID(ctx context.Context, id string) (*variationdomain.Variation, error)
	// List lists variations.
	List(ctx context.Context) ([]variationdomain.Variation, error)
	// Update persists variation updates.
	Update(ctx context.Context, variation *variationdomain.Variation) error
	// Delete deletes variations by ID.
	Delete(ctx context.Context, id string) error
}
