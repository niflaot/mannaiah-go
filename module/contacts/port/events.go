package port

import (
	"context"
	"time"
)

// IntegrationEvent defines transport-level contact integration event envelopes.
type IntegrationEvent struct {
	// ID defines unique event identifiers.
	ID string
	// Topic defines event routing topics.
	Topic string
	// SchemaVersion defines payload schema versions.
	SchemaVersion string
	// OccurredAt defines event timestamps.
	OccurredAt time.Time
	// CorrelationID defines end-to-end flow correlation values.
	CorrelationID string
	// CausationID defines causal event references.
	CausationID string
	// Payload defines serialized event payload values.
	Payload any
	// Metadata defines optional transport metadata values.
	Metadata map[string]string
}

// IntegrationEventPublisher defines transport publication behavior for integration events.
type IntegrationEventPublisher interface {
	// Publish emits integration events to the configured transport.
	Publish(ctx context.Context, event IntegrationEvent) error
}
