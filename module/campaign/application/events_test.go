package application

import (
	"context"
	"testing"
	"time"

	"mannaiah/module/campaign/port"
)

type integrationEventPublisherMock struct {
	publishFn func(ctx context.Context, event port.IntegrationEvent) error
}

func (m integrationEventPublisherMock) Publish(ctx context.Context, event port.IntegrationEvent) error {
	return m.publishFn(ctx, event)
}

// TestBuildCampaignDeliveryIntegrationEvent verifies campaign delivery event mapping behavior.
func TestBuildCampaignDeliveryIntegrationEvent(t *testing.T) {
	now := time.Now().UTC()
	event := buildCampaignDeliveryIntegrationEvent("cmp-1", "c-1", "email", "submitted_to_provider", 1, now)
	if event.Topic != port.TopicCampaignDelivery {
		t.Fatalf("event.Topic = %q, want %q", event.Topic, port.TopicCampaignDelivery)
	}
	if event.ID == "" {
		t.Fatalf("expected non-empty event id")
	}
	payload, ok := event.Payload.(port.CampaignDeliveryPayload)
	if !ok {
		t.Fatalf("event.Payload type = %T, want CampaignDeliveryPayload", event.Payload)
	}
	if payload.CampaignID != "cmp-1" || payload.ContactID != "c-1" {
		t.Fatalf("payload ids = %q/%q, want cmp-1/c-1", payload.CampaignID, payload.ContactID)
	}
}

// TestResolvePublisher verifies optional campaign publisher resolution behavior.
func TestResolvePublisher(t *testing.T) {
	publisher := resolvePublisher(nil)
	if err := publisher.Publish(context.Background(), port.IntegrationEvent{}); err != nil {
		t.Fatalf("noop publisher error = %v", err)
	}

	called := false
	resolved := resolvePublisher(integrationEventPublisherMock{publishFn: func(ctx context.Context, event port.IntegrationEvent) error {
		called = true
		return nil
	}})
	if err := resolved.Publish(context.Background(), port.IntegrationEvent{}); err != nil {
		t.Fatalf("resolved publisher error = %v", err)
	}
	if !called {
		t.Fatalf("expected resolved publisher invocation")
	}
}
