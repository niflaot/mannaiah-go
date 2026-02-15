package event

import (
	"context"
	"errors"
	"testing"
	"time"

	"mannaiah/module/core/messaging/bus"
	ordersport "mannaiah/module/orders/port"
)

// busPublisherMock defines bus publication behavior for publisher tests.
type busPublisherMock struct {
	// message defines captured bus message values.
	message bus.Message
	// err defines publication errors.
	err error
}

// Publish captures bus message values.
func (m *busPublisherMock) Publish(ctx context.Context, message bus.Message) error {
	m.message = message
	return m.err
}

// TestNewPublisherValidation verifies constructor validation behavior.
func TestNewPublisherValidation(t *testing.T) {
	if _, err := NewPublisher(nil); !errors.Is(err, ErrNilPublisher) {
		t.Fatalf("NewPublisher(nil) error = %v, want ErrNilPublisher", err)
	}
}

// TestPublish verifies integration event publication mapping behavior.
func TestPublish(t *testing.T) {
	mock := &busPublisherMock{}
	publisher, err := NewPublisher(mock)
	if err != nil {
		t.Fatalf("NewPublisher() error = %v", err)
	}

	now := time.Date(2026, time.February, 15, 12, 0, 0, 0, time.UTC)
	err = publisher.Publish(context.Background(), ordersport.IntegrationEvent{
		ID:            "evt-1",
		Topic:         ordersport.TopicOrderUpdated,
		SchemaVersion: "v1",
		OccurredAt:    now,
		CorrelationID: "cor-1",
		CausationID:   "cau-1",
		Payload: map[string]any{
			"id": "o-1",
		},
		Metadata: map[string]string{
			"aggregate_id": "o-1",
		},
	})
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	if mock.message.Topic != ordersport.TopicOrderUpdated {
		t.Fatalf("message.Topic = %q, want %q", mock.message.Topic, ordersport.TopicOrderUpdated)
	}
	if mock.message.Metadata[bus.MetadataSchemaVersion] != "v1" {
		t.Fatalf("schema metadata = %q, want %q", mock.message.Metadata[bus.MetadataSchemaVersion], "v1")
	}
	if mock.message.Metadata[bus.MetadataCorrelationID] != "cor-1" {
		t.Fatalf("correlation metadata = %q, want %q", mock.message.Metadata[bus.MetadataCorrelationID], "cor-1")
	}
}

// TestPublishError verifies publication error mapping behavior.
func TestPublishError(t *testing.T) {
	mock := &busPublisherMock{err: errors.New("publish failed")}
	publisher, err := NewPublisher(mock)
	if err != nil {
		t.Fatalf("NewPublisher() error = %v", err)
	}

	if err := publisher.Publish(context.Background(), ordersport.IntegrationEvent{
		ID:            "evt-1",
		Topic:         ordersport.TopicOrderUpdated,
		SchemaVersion: "v1",
		OccurredAt:    time.Now().UTC(),
		Payload:       map[string]any{"id": "o-1"},
	}); err == nil {
		t.Fatalf("expected publish error")
	}
}

