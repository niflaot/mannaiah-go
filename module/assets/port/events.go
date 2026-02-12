package port

import (
	"context"
	"time"
)

// IntegrationEvent defines module-level integration event envelope values.
type IntegrationEvent struct {
	// ID defines unique integration event identifiers.
	ID string
	// Topic defines event topic values.
	Topic string
	// Payload defines serializable event payload values.
	Payload any
	// CorrelationID defines correlation identifiers.
	CorrelationID string
	// CausationID defines causation identifiers.
	CausationID string
	// SchemaVersion defines payload schema versions.
	SchemaVersion string
	// OccurredAt defines event occurrence timestamps.
	OccurredAt time.Time
	// Metadata defines extra integration metadata values.
	Metadata map[string]string
}

// IntegrationEventPublisher defines integration event publication behavior.
type IntegrationEventPublisher interface {
	// Publish publishes integration events.
	Publish(ctx context.Context, event IntegrationEvent) error
}
