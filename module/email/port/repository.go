package port

import (
	"context"
	"mannaiah/module/email/domain"
)

// Repository defines email delivery persistence behavior.
type Repository interface {
	// CreateDelivery persists delivery rows.
	CreateDelivery(ctx context.Context, delivery *domain.Delivery) error
	// UpdateDeliveryStatus updates current delivery status values.
	UpdateDeliveryStatus(ctx context.Context, deliveryID string, status domain.DeliveryStatus, providerMessageID string) error
	// AddStatusEntry persists immutable status timeline rows.
	AddStatusEntry(ctx context.Context, entry *domain.StatusEntry) error
	// CountStatusEntries counts status timeline rows for one delivery/status pair.
	CountStatusEntries(ctx context.Context, deliveryID string, status domain.DeliveryStatus) (int64, error)
	// GetByID retrieves delivery rows by id.
	GetByID(ctx context.Context, id string) (*domain.Delivery, error)
	// GetByProviderMessageID retrieves delivery rows by provider message id.
	GetByProviderMessageID(ctx context.Context, providerMessageID string) (*domain.Delivery, error)
	// ListByEmail retrieves all deliveries sent to one recipient email.
	ListByEmail(ctx context.Context, email string) ([]*domain.Delivery, error)
	// ListByCampaignID retrieves paginated delivery rows for a campaign by idempotency key prefix.
	ListByCampaignID(ctx context.Context, campaignID string, page int, limit int) ([]*domain.Delivery, int64, error)
}
