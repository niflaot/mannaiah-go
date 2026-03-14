package port

import (
	"context"
	"time"
)

const (
	// TopicCampaignDelivery defines campaign delivery integration event topics.
	TopicCampaignDelivery = "campaign.v1.delivery"
)

// CampaignDeliveryPayload defines campaign delivery event payload values.
type CampaignDeliveryPayload struct {
	// CampaignID defines campaign identifier values.
	CampaignID string `json:"campaignId"`
	// ContactID defines contact identifier values.
	ContactID string `json:"contactId"`
	// Channel defines channel values.
	Channel string `json:"channel"`
	// Status defines delivery status values.
	Status string `json:"status"`
	// TemplateVersion defines campaign template version values.
	TemplateVersion int `json:"templateVersion"`
	// OccurredAt defines event timestamps.
	OccurredAt time.Time `json:"occurredAt"`
}

// IntegrationEvent defines transport-level campaign integration event envelopes.
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

// IntegrationEventPublisher defines event publication behavior for campaign integration events.
type IntegrationEventPublisher interface {
	// Publish emits integration events to the configured transport.
	Publish(ctx context.Context, event IntegrationEvent) error
}
