package event

import (
	"context"
	"errors"
	"testing"
	"time"

	"mannaiah/module/campaign/port"
	"mannaiah/module/core/messaging/bus"
)

type busPublisherMock struct {
	publishFn func(ctx context.Context, message bus.Message) error
}

func (m busPublisherMock) Publish(ctx context.Context, message bus.Message) error {
	return m.publishFn(ctx, message)
}

// TestNewPublisherRejectsNil verifies constructor validation behavior.
func TestNewPublisherRejectsNil(t *testing.T) {
	if _, err := NewPublisher(nil); !errors.Is(err, ErrNilPublisher) {
		t.Fatalf("NewPublisher(nil) error = %v, want ErrNilPublisher", err)
	}
}

// TestPublish verifies integration event publication mapping behavior.
func TestPublish(t *testing.T) {
	mock := busPublisherMock{publishFn: func(ctx context.Context, message bus.Message) error {
		if message.Topic != port.TopicCampaignDelivery {
			t.Fatalf("message.Topic = %q, want %q", message.Topic, port.TopicCampaignDelivery)
		}
		if message.ID != "evt-1" {
			t.Fatalf("message.ID = %q, want %q", message.ID, "evt-1")
		}
		if message.Metadata[bus.MetadataSchemaVersion] != "v1" {
			t.Fatalf("schema version = %q, want %q", message.Metadata[bus.MetadataSchemaVersion], "v1")
		}
		return nil
	}}

	publisher, err := NewPublisher(mock)
	if err != nil {
		t.Fatalf("NewPublisher() error = %v", err)
	}

	err = publisher.Publish(context.Background(), port.IntegrationEvent{
		ID:            "evt-1",
		Topic:         port.TopicCampaignDelivery,
		SchemaVersion: "v1",
		OccurredAt:    time.Now().UTC(),
		Payload:       port.CampaignDeliveryPayload{CampaignID: "cmp-1", ContactID: "c-1", Channel: "email", Status: "sent", TemplateVersion: 1, OccurredAt: time.Now().UTC()},
		Metadata:      map[string]string{"campaign_id": "cmp-1"},
	})
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
}
