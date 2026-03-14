package port

import (
	"context"
	"mannaiah/module/segment/domain"
)

// Repository defines segment persistence behavior.
type Repository interface {
	// Create persists segment rows.
	Create(ctx context.Context, segment *domain.Segment) error
	// GetByID retrieves one segment by id.
	GetByID(ctx context.Context, id string) (*domain.Segment, error)
	// List retrieves segment rows.
	List(ctx context.Context, page int, limit int) ([]domain.Segment, int64, error)
	// Update persists segment row updates.
	Update(ctx context.Context, segment *domain.Segment) error
	// Delete removes one segment by id.
	Delete(ctx context.Context, id string) error
}
