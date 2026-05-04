package port

import (
	"context"
	"time"
)

// IntegrationEvent defines transport-level WooCommerce integration event envelopes.
type IntegrationEvent struct {
	// ID defines unique event identifiers.
	ID string
	// Topic defines event routing topics.
	Topic string
	// SchemaVersion defines payload schema versions.
	SchemaVersion string
	// OccurredAt defines event timestamps.
	OccurredAt time.Time
	// Payload defines serialized event payload values.
	Payload any
	// Metadata defines optional transport metadata values.
	Metadata map[string]string
}

// IntegrationEventPublisher defines event publication behavior for WooCommerce integration events.
type IntegrationEventPublisher interface {
	// Publish emits integration events to the configured transport.
	Publish(ctx context.Context, event IntegrationEvent) error
}
