package port

import (
	"context"
	"time"
)

const (
	// TopicMarkGenerated defines mark-generated integration event topics.
	TopicMarkGenerated = "shipping.v1.mark.generated"
	// TopicMarkFailed defines mark-failed integration event topics.
	TopicMarkFailed = "shipping.v1.mark.failed"
	// TopicMarkVoided defines mark-voided integration event topics.
	TopicMarkVoided = "shipping.v1.mark.voided"
	// TopicBatchCreated defines batch-created integration event topics.
	TopicBatchCreated = "shipping.v1.batch.created"
	// TopicBatchClosed defines batch-closed integration event topics.
	TopicBatchClosed = "shipping.v1.batch.closed"
	// TopicTrackingUpdated defines tracking-updated integration event topics.
	TopicTrackingUpdated = "shipping.v1.tracking.updated"
)

// IntegrationEvent defines transport-level integration event values.
type IntegrationEvent struct {
	// ID defines unique event identifier values.
	ID string
	// Topic defines routing topic values.
	Topic string
	// SchemaVersion defines payload schema-version values.
	SchemaVersion string
	// OccurredAt defines event occurrence timestamps.
	OccurredAt time.Time
	// Payload defines event payload values.
	Payload any
	// Metadata defines optional transport metadata values.
	Metadata map[string]string
}

// IntegrationEventPublisher defines integration event publication behavior.
type IntegrationEventPublisher interface {
	// Publish publishes one integration event to transport.
	Publish(ctx context.Context, event IntegrationEvent) error
}
