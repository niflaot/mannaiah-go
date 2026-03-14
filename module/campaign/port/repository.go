package port

import (
	"context"
	"mannaiah/module/campaign/domain"
)

// Repository defines campaign persistence behavior.
type Repository interface {
	// Create persists campaign rows.
	Create(ctx context.Context, campaign *domain.Campaign) error
	// GetByID retrieves campaign rows by id.
	GetByID(ctx context.Context, id string) (*domain.Campaign, error)
	// List retrieves campaign rows.
	List(ctx context.Context, page int, limit int) ([]domain.Campaign, int64, error)
	// Update persists campaign row updates.
	Update(ctx context.Context, campaign *domain.Campaign) error
	// Delete removes one campaign by id.
	Delete(ctx context.Context, id string) error
}
