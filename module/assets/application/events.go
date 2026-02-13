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
	// TopicFolderCreated defines folder-created integration event topics.
	TopicFolderCreated = "asset_folders.v1.created"
	// TopicFolderUpdated defines folder-updated integration event topics.
	TopicFolderUpdated = "asset_folders.v1.updated"
	// TopicFolderDeleted defines folder-deleted integration event topics.
	TopicFolderDeleted = "asset_folders.v1.deleted"
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
	// FolderID defines optional logical folder identifiers.
	FolderID string `json:"folderId,omitempty"`
	// MimeType defines payload mime types.
	MimeType string `json:"mimeType"`
	// Size defines payload size in bytes.
	Size int64 `json:"size"`
	// Tags defines optional classification tags.
	Tags []domain.Tag `json:"tags,omitempty"`
	// Metadata defines optional key-value metadata values.
	Metadata map[string]string `json:"metadata,omitempty"`
	// IsDeleted reports soft-delete status.
	IsDeleted bool `json:"isDeleted"`
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time `json:"updatedAt"`
}

// FolderEventPayload defines integration event payload values for folder lifecycle events.
type FolderEventPayload struct {
	// ID defines folder identifiers.
	ID string `json:"id"`
	// Name defines folder names.
	Name string `json:"name"`
	// Slug defines normalized folder slugs.
	Slug string `json:"slug"`
	// Tags defines optional classification tags.
	Tags []domain.Tag `json:"tags,omitempty"`
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

// buildFolderCreatedIntegrationEvent maps created folders into integration events.
func buildFolderCreatedIntegrationEvent(folder domain.Folder) port.IntegrationEvent {
	return buildFolderIntegrationEvent(TopicFolderCreated, folder)
}

// buildFolderUpdatedIntegrationEvent maps updated folders into integration events.
func buildFolderUpdatedIntegrationEvent(folder domain.Folder) port.IntegrationEvent {
	return buildFolderIntegrationEvent(TopicFolderUpdated, folder)
}

// buildFolderDeletedIntegrationEvent maps deleted folders into integration events.
func buildFolderDeletedIntegrationEvent(folder domain.Folder) port.IntegrationEvent {
	payload := toFolderEventPayload(folder)
	payload.IsDeleted = true

	return port.IntegrationEvent{
		ID:            generateEventID(),
		Topic:         TopicFolderDeleted,
		SchemaVersion: assetSchemaVersionV1,
		OccurredAt:    time.Now().UTC(),
		Payload:       payload,
		Metadata: map[string]string{
			"aggregate_id": strings.TrimSpace(folder.ID),
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

// buildFolderIntegrationEvent maps folders into common integration event envelopes.
func buildFolderIntegrationEvent(topic string, folder domain.Folder) port.IntegrationEvent {
	occurredAt := time.Now().UTC()

	return port.IntegrationEvent{
		ID:            generateEventID(),
		Topic:         topic,
		SchemaVersion: assetSchemaVersionV1,
		OccurredAt:    occurredAt,
		Payload:       toFolderEventPayload(folder),
		Metadata: map[string]string{
			"aggregate_id": strings.TrimSpace(folder.ID),
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
		FolderID:     strings.TrimSpace(asset.FolderID),
		MimeType:     strings.TrimSpace(asset.MimeType),
		Size:         asset.Size,
		Tags:         asset.Tags,
		Metadata:     asset.Metadata,
		IsDeleted:    asset.IsDeleted,
		CreatedAt:    asset.CreatedAt,
		UpdatedAt:    asset.UpdatedAt,
	}
}

// toFolderEventPayload maps folder entities into integration payload values.
func toFolderEventPayload(folder domain.Folder) FolderEventPayload {
	return FolderEventPayload{
		ID:        strings.TrimSpace(folder.ID),
		Name:      strings.TrimSpace(folder.Name),
		Slug:      strings.TrimSpace(folder.Slug),
		Tags:      folder.Tags,
		IsDeleted: folder.IsDeleted,
		CreatedAt: folder.CreatedAt,
		UpdatedAt: folder.UpdatedAt,
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
