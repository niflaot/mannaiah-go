package event

import (
	"context"
	"encoding/json"
	errorspkg "errors"
	"testing"
	"time"

	"mannaiah/module/core/messaging/bus"
	"mannaiah/module/woocommerce/port"
)

// busPublisherMock defines bus publication behavior for publisher tests.
type busPublisherMock struct {
	// message defines captured published messages.
	message bus.Message
	// err defines forced publication errors.
	err error
}

// Publish emits messages.
func (m *busPublisherMock) Publish(ctx context.Context, message bus.Message) error {
	if m.err != nil {
		return m.err
	}

	m.message = message
	return nil
}

// TestNewPublisherValidation verifies constructor validation behavior.
func TestNewPublisherValidation(t *testing.T) {
	if _, err := NewPublisher(nil); !errorspkg.Is(err, ErrNilPublisher) {
		t.Fatalf("NewPublisher(nil) error = %v, want ErrNilPublisher", err)
	}
}

// TestPublish verifies payload and metadata mapping behavior.
func TestPublish(t *testing.T) {
	mock := &busPublisherMock{}
	publisher, err := NewPublisher(mock)
	if err != nil {
		t.Fatalf("NewPublisher() error = %v", err)
	}

	event := port.IntegrationEvent{
		ID:            "event-1",
		Topic:         "topic-1",
		SchemaVersion: "v1",
		OccurredAt:    time.Unix(100, 0).UTC(),
		Payload: map[string]any{
			"status": "ok",
		},
		Metadata: map[string]string{
			"source": "test",
		},
	}

	if err := publisher.Publish(context.Background(), event); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if mock.message.Topic != event.Topic {
		t.Fatalf("message.Topic = %q, want %q", mock.message.Topic, event.Topic)
	}
	if mock.message.Metadata[bus.MetadataSchemaVersion] != "v1" {
		t.Fatalf("schema version metadata mismatch")
	}

	var payload map[string]any
	if err := json.Unmarshal(mock.message.Payload, &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload["status"] != "ok" {
		t.Fatalf("payload.status = %v, want %q", payload["status"], "ok")
	}
}

// TestPublishError verifies publication error propagation behavior.
func TestPublishError(t *testing.T) {
	mock := &busPublisherMock{err: errorspkg.New("publish failed")}
	publisher, err := NewPublisher(mock)
	if err != nil {
		t.Fatalf("NewPublisher() error = %v", err)
	}

	if err := publisher.Publish(context.Background(), port.IntegrationEvent{
		ID:            "event-1",
		Topic:         "topic-1",
		SchemaVersion: "v1",
		OccurredAt:    time.Now().UTC(),
		Payload:       map[string]any{"status": "ok"},
	}); err == nil {
		t.Fatalf("expected Publish() error")
	}
}

// TestPublishMarshalError verifies payload marshal error handling behavior.
func TestPublishMarshalError(t *testing.T) {
	mock := &busPublisherMock{}
	publisher, err := NewPublisher(mock)
	if err != nil {
		t.Fatalf("NewPublisher() error = %v", err)
	}

	if err := publisher.Publish(context.Background(), port.IntegrationEvent{
		ID:            "event-1",
		Topic:         "topic-1",
		SchemaVersion: "v1",
		OccurredAt:    time.Now().UTC(),
		Payload:       map[string]any{"invalid": make(chan int)},
	}); err == nil {
		t.Fatalf("expected marshal error")
	}
}
