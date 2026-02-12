package event

import (
	"context"
	errorspkg "errors"
	"testing"
	"time"

	"mannaiah/module/assets/port"
	"mannaiah/module/core/messaging/bus"
)

// busPublisherMock defines core bus publication behavior for tests.
type busPublisherMock struct {
	// publishFn defines publish behavior.
	publishFn func(ctx context.Context, message bus.Message) error
}

// Publish executes configured publish behavior.
func (m busPublisherMock) Publish(ctx context.Context, message bus.Message) error {
	return m.publishFn(ctx, message)
}

// TestNewPublisher verifies constructor validation behavior.
func TestNewPublisher(t *testing.T) {
	if _, err := NewPublisher(nil); !errorspkg.Is(err, ErrNilPublisher) {
		t.Fatalf("NewPublisher(nil) error = %v, want ErrNilPublisher", err)
	}
}

// TestPublish verifies integration message publication mapping behavior.
func TestPublish(t *testing.T) {
	var captured bus.Message
	publisher, err := NewPublisher(busPublisherMock{publishFn: func(ctx context.Context, message bus.Message) error {
		captured = message
		return nil
	}})
	if err != nil {
		t.Fatalf("NewPublisher() error = %v", err)
	}

	event := port.IntegrationEvent{
		ID:            "evt-1",
		Topic:         "assets.v1.created",
		SchemaVersion: "v1",
		OccurredAt:    time.Unix(10, 0).UTC(),
		CorrelationID: "corr-1",
		CausationID:   "cause-1",
		Payload: map[string]any{
			"id": "asset-1",
		},
		Metadata: map[string]string{"k": "v"},
	}

	if publishErr := publisher.Publish(context.Background(), event); publishErr != nil {
		t.Fatalf("Publish() error = %v", publishErr)
	}
	if captured.ID != event.ID {
		t.Fatalf("captured.ID = %q, want %q", captured.ID, event.ID)
	}
	if captured.Topic != event.Topic {
		t.Fatalf("captured.Topic = %q, want %q", captured.Topic, event.Topic)
	}
	if captured.Metadata[bus.MetadataCorrelationID] != event.CorrelationID {
		t.Fatalf("missing correlation metadata")
	}
}

// TestPublishMarshalFailure verifies payload marshalling error behavior.
func TestPublishMarshalFailure(t *testing.T) {
	publisher, err := NewPublisher(busPublisherMock{publishFn: func(ctx context.Context, message bus.Message) error { return nil }})
	if err != nil {
		t.Fatalf("NewPublisher() error = %v", err)
	}

	marshalErr := publisher.Publish(context.Background(), port.IntegrationEvent{
		ID:      "evt-1",
		Topic:   "assets.v1.created",
		Payload: map[string]any{"fn": func() {}},
	})
	if marshalErr == nil {
		t.Fatalf("expected marshal error")
	}
}

// TestPublishTransportFailure verifies transport publication error behavior.
func TestPublishTransportFailure(t *testing.T) {
	transportErr := errorspkg.New("transport failed")
	publisher, err := NewPublisher(busPublisherMock{publishFn: func(ctx context.Context, message bus.Message) error {
		return transportErr
	}})
	if err != nil {
		t.Fatalf("NewPublisher() error = %v", err)
	}

	publishErr := publisher.Publish(context.Background(), port.IntegrationEvent{
		ID:      "evt-1",
		Topic:   "assets.v1.created",
		Payload: map[string]any{"id": "a-1"},
	})
	if !errorspkg.Is(publishErr, transportErr) {
		t.Fatalf("Publish() error = %v, want transportErr", publishErr)
	}
}
