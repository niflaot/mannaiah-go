package port

import (
	"context"
	"time"
)

const (
	// TopicMembershipChanged defines membership-changed integration event topics.
	TopicMembershipChanged = "membership.v1.changed"
)

// MembershipChangedPayload defines membership change event payload values.
type MembershipChangedPayload struct {
	// ContactID defines contact identifier values.
	ContactID string `json:"contactId"`
	// Channel defines channel values.
	Channel string `json:"channel"`
	// Action defines action values.
	Action string `json:"action"`
	// Source defines source values.
	Source string `json:"source"`
	// OccurredAt defines action timestamp values.
	OccurredAt time.Time `json:"occurredAt"`
}

// IntegrationEvent defines transport-level membership integration event envelopes.
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

// IntegrationEventPublisher defines event publication behavior for membership integration events.
type IntegrationEventPublisher interface {
	// Publish emits integration events to the configured transport.
	Publish(ctx context.Context, event IntegrationEvent) error
}
