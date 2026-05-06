package port

import "context"

// IntegrationEventPublisher defines optional Shopify integration event publication behavior.
type IntegrationEventPublisher interface {
	// Publish emits one integration event payload.
	Publish(ctx context.Context, event any) error
}
