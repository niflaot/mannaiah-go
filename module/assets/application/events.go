package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"mannaiah/module/assets/domain"
	"mannaiah/module/assets/port"
)

const (
	// TopicAssetCreated defines asset-created integration event topics.
	TopicAssetCreated = "assets.v1.created"
	// TopicAssetUpdated defines asset-updated integration event topics.
	TopicAssetUpdated = "assets.v1.updated"
	// TopicAssetDeleted defines asset-deleted integration event topics.
	TopicAssetDeleted = "assets.v1.deleted"
	// assetSchemaVersionV1 defines integration event schema versions.
	assetSchemaVersionV1 = "v1"
)

// AssetEventPayload defines integration event payload values for asset lifecycle events.
type AssetEventPayload struct {
	// ID defines asset identifiers.
	ID string `json:"id"`
	// Key defines storage object keys.
	Key string `json:"key"`
	// Name defines asset display names.
	Name string `json:"name"`
	// OriginalName defines uploaded file names.
	OriginalName string `json:"originalName"`
	// MimeType defines payload mime types.
	MimeType string `json:"mimeType"`
	// Size defines payload size in bytes.
	Size int64 `json:"size"`
	// IsDeleted reports soft-delete status.
	IsDeleted bool `json:"isDeleted"`
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time `json:"updatedAt"`
}

// noopIntegrationEventPublisher defines no-op event publication behavior.
type noopIntegrationEventPublisher struct{}

// Publish ignores integration event publication requests.
func (noopIntegrationEventPublisher) Publish(ctx context.Context, event port.IntegrationEvent) error {
	return nil
}

// resolvePublisher resolves optional publisher dependencies.
func resolvePublisher(publishers []port.IntegrationEventPublisher) port.IntegrationEventPublisher {
	if len(publishers) == 0 || publishers[0] == nil {
		return noopIntegrationEventPublisher{}
	}

	return publishers[0]
}

// buildAssetCreatedIntegrationEvent maps created assets into integration events.
func buildAssetCreatedIntegrationEvent(asset domain.Asset) port.IntegrationEvent {
	return buildIntegrationEvent(TopicAssetCreated, asset)
}

// buildAssetUpdatedIntegrationEvent maps updated assets into integration events.
func buildAssetUpdatedIntegrationEvent(asset domain.Asset) port.IntegrationEvent {
	return buildIntegrationEvent(TopicAssetUpdated, asset)
}

// buildAssetDeletedIntegrationEvent maps deleted assets into integration events.
func buildAssetDeletedIntegrationEvent(asset domain.Asset) port.IntegrationEvent {
	payload := toAssetEventPayload(asset)
	payload.IsDeleted = true

	return port.IntegrationEvent{
		ID:            generateEventID(),
		Topic:         TopicAssetDeleted,
		SchemaVersion: assetSchemaVersionV1,
		OccurredAt:    time.Now().UTC(),
		Payload:       payload,
		Metadata: map[string]string{
			"aggregate_id": strings.TrimSpace(asset.ID),
		},
	}
}

// buildIntegrationEvent maps assets into common integration event envelopes.
func buildIntegrationEvent(topic string, asset domain.Asset) port.IntegrationEvent {
	occurredAt := time.Now().UTC()

	return port.IntegrationEvent{
		ID:            generateEventID(),
		Topic:         topic,
		SchemaVersion: assetSchemaVersionV1,
		OccurredAt:    occurredAt,
		Payload:       toAssetEventPayload(asset),
		Metadata: map[string]string{
			"aggregate_id": strings.TrimSpace(asset.ID),
		},
	}
}

// toAssetEventPayload maps asset entities into integration payload values.
func toAssetEventPayload(asset domain.Asset) AssetEventPayload {
	return AssetEventPayload{
		ID:           strings.TrimSpace(asset.ID),
		Key:          strings.TrimSpace(asset.Key),
		Name:         strings.TrimSpace(asset.Name),
		OriginalName: strings.TrimSpace(asset.OriginalName),
		MimeType:     strings.TrimSpace(asset.MimeType),
		Size:         asset.Size,
		IsDeleted:    asset.IsDeleted,
		CreatedAt:    asset.CreatedAt,
		UpdatedAt:    asset.UpdatedAt,
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
