package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"mannaiah/module/campaign/port"
)

const (
	campaignSchemaVersionV1 = "v1"
)

// noopIntegrationEventPublisher defines no-op campaign event publishing behavior.
type noopIntegrationEventPublisher struct{}

// Publish ignores integration events.
func (noopIntegrationEventPublisher) Publish(ctx context.Context, event port.IntegrationEvent) error {
	return nil
}

// resolvePublisher resolves optional integration event publisher dependencies.
func resolvePublisher(publisher port.IntegrationEventPublisher) port.IntegrationEventPublisher {
	if publisher != nil {
		return publisher
	}

	return noopIntegrationEventPublisher{}
}

// buildCampaignDeliveryIntegrationEvent builds campaign delivery integration events.
func buildCampaignDeliveryIntegrationEvent(campaignID string, contactID string, channel string, status string, templateVersion int, occurredAt time.Time) port.IntegrationEvent {
	resolvedOccurredAt := occurredAt.UTC()
	if resolvedOccurredAt.IsZero() {
		resolvedOccurredAt = time.Now().UTC()
	}

	payload := port.CampaignDeliveryPayload{
		CampaignID:      strings.TrimSpace(campaignID),
		ContactID:       strings.TrimSpace(contactID),
		Channel:         strings.TrimSpace(channel),
		Status:          strings.TrimSpace(status),
		TemplateVersion: templateVersion,
		OccurredAt:      resolvedOccurredAt,
	}

	return port.IntegrationEvent{
		ID:            generateEventID(),
		Topic:         port.TopicCampaignDelivery,
		SchemaVersion: campaignSchemaVersionV1,
		OccurredAt:    resolvedOccurredAt,
		Payload:       payload,
		Metadata: map[string]string{
			"campaign_id": payload.CampaignID,
			"contact_id":  payload.ContactID,
			"status":      payload.Status,
		},
	}
}

// generateEventID creates random integration event identifiers.
func generateEventID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("event-%d", time.Now().UnixNano())
	}

	return strings.TrimSpace(hex.EncodeToString(bytes))
}
