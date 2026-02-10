package event

import (
	"context"
	"errors"
	"testing"
	"time"

	"mannaiah/module/contacts/port"
	"mannaiah/module/core/messaging/bus"
)

// busPublisherMock defines core message bus publish behavior for unit tests.
type busPublisherMock struct {
	// publishFn defines publish behavior.
	publishFn func(ctx context.Context, message bus.Message) error
}

// Publish executes configured publish behavior.
func (m busPublisherMock) Publish(ctx context.Context, message bus.Message) error {
	return m.publishFn(ctx, message)
}

// TestNewPublisherRejectsNil verifies constructor validation.
func TestNewPublisherRejectsNil(t *testing.T) {
	_, err := NewPublisher(nil)
	if !errors.Is(err, ErrNilPublisher) {
		t.Fatalf("NewPublisher() error = %v, want ErrNilPublisher", err)
	}
}

// TestPublishMapsIntegrationEvents verifies integration event mapping into bus messages.
func TestPublishMapsIntegrationEvents(t *testing.T) {
	called := false
	publisher, err := NewPublisher(busPublisherMock{publishFn: func(ctx context.Context, message bus.Message) error {
		called = true
		if message.Topic != "contacts.v1.created" {
			t.Fatalf("Topic = %q, want %q", message.Topic, "contacts.v1.created")
		}
		if message.ID != "evt-1" {
			t.Fatalf("ID = %q, want %q", message.ID, "evt-1")
		}
		if message.Metadata[bus.MetadataSchemaVersion] != "v1" {
			t.Fatalf("schema_version = %q, want %q", message.Metadata[bus.MetadataSchemaVersion], "v1")
		}
		if message.Metadata[bus.MetadataCorrelationID] != "corr-1" {
			t.Fatalf("correlation_id = %q, want %q", message.Metadata[bus.MetadataCorrelationID], "corr-1")
		}

		return nil
	}})
	if err != nil {
		t.Fatalf("NewPublisher() error = %v", err)
	}

	publishErr := publisher.Publish(context.Background(), port.IntegrationEvent{
		ID:            "evt-1",
		Topic:         "contacts.v1.created",
		SchemaVersion: "v1",
		OccurredAt:    time.Now().UTC(),
		CorrelationID: "corr-1",
		Payload: map[string]any{
			"id": "c-1",
		},
		Metadata: map[string]string{"aggregate_id": "c-1"},
	})
	if publishErr != nil {
		t.Fatalf("Publish() error = %v", publishErr)
	}
	if !called {
		t.Fatalf("expected publish call")
	}
}

// TestPublishPropagatesErrors verifies payload and transport error propagation.
func TestPublishPropagatesErrors(t *testing.T) {
	publisher, err := NewPublisher(busPublisherMock{publishFn: func(ctx context.Context, message bus.Message) error {
		return errors.New("transport down")
	}})
	if err != nil {
		t.Fatalf("NewPublisher() error = %v", err)
	}

	publishErr := publisher.Publish(context.Background(), port.IntegrationEvent{
		ID:            "evt-1",
		Topic:         "contacts.v1.created",
		SchemaVersion: "v1",
		OccurredAt:    time.Now().UTC(),
		Payload:       map[string]any{"id": "c-1"},
	})
	if publishErr == nil {
		t.Fatalf("expected publish error")
	}

	marshalErr := publisher.Publish(context.Background(), port.IntegrationEvent{
		ID:            "evt-2",
		Topic:         "contacts.v1.updated",
		SchemaVersion: "v1",
		OccurredAt:    time.Now().UTC(),
		Payload:       map[string]any{"broken": make(chan int)},
	})
	if marshalErr == nil {
		t.Fatalf("expected marshal error")
	}
}
