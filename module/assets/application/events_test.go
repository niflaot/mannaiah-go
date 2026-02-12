package application

import (
	"context"
	testing "testing"
	"time"

	"mannaiah/module/assets/domain"
	"mannaiah/module/assets/port"
)

// TestResolvePublisher verifies publisher fallback behavior.
func TestResolvePublisher(t *testing.T) {
	resolved := resolvePublisher(nil)
	if resolved == nil {
		t.Fatalf("resolvePublisher(nil) returned nil")
	}
	if err := resolved.Publish(context.Background(), port.IntegrationEvent{}); err != nil {
		t.Fatalf("fallback publisher error = %v", err)
	}

	custom := noopIntegrationEventPublisher{}
	if resolvePublisher([]port.IntegrationEventPublisher{custom}) == nil {
		t.Fatalf("resolvePublisher(custom) returned nil")
	}
}

// TestBuildIntegrationEvents verifies event topic and payload mapping behavior.
func TestBuildIntegrationEvents(t *testing.T) {
	asset := domain.Asset{
		ID:           "a-1",
		Key:          "assets/a-1.png",
		Name:         "Asset",
		OriginalName: "a.png",
		MimeType:     "image/png",
		Size:         128,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	created := buildAssetCreatedIntegrationEvent(asset)
	if created.Topic != TopicAssetCreated {
		t.Fatalf("created.Topic = %q, want %q", created.Topic, TopicAssetCreated)
	}
	if created.SchemaVersion != assetSchemaVersionV1 {
		t.Fatalf("created.SchemaVersion = %q, want %q", created.SchemaVersion, assetSchemaVersionV1)
	}
	if created.Payload == nil {
		t.Fatalf("expected created payload")
	}

	updated := buildAssetUpdatedIntegrationEvent(asset)
	if updated.Topic != TopicAssetUpdated {
		t.Fatalf("updated.Topic = %q, want %q", updated.Topic, TopicAssetUpdated)
	}

	deleted := buildAssetDeletedIntegrationEvent(asset)
	if deleted.Topic != TopicAssetDeleted {
		t.Fatalf("deleted.Topic = %q, want %q", deleted.Topic, TopicAssetDeleted)
	}
	payload, ok := deleted.Payload.(AssetEventPayload)
	if !ok {
		t.Fatalf("deleted payload type mismatch")
	}
	if !payload.IsDeleted {
		t.Fatalf("deleted payload IsDeleted = false, want true")
	}
}

// TestGenerateEventID verifies non-empty event-id generation behavior.
func TestGenerateEventID(t *testing.T) {
	value := generateEventID()
	if value == "" {
		t.Fatalf("generateEventID() returned empty value")
	}
}
