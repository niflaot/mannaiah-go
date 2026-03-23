package event

import (
	"context"
	"testing"
	"time"

	"mannaiah/module/core/messaging/bus"
	"mannaiah/module/shipping/port"
)

type busPublisherStub struct {
	messages []bus.Message
}

func (s *busPublisherStub) Publish(ctx context.Context, message bus.Message) error {
	s.messages = append(s.messages, message)

	return nil
}

// TestPublish verifies integration event publication behavior.
func TestPublish(t *testing.T) {
	publisherStub := &busPublisherStub{}
	publisher, err := NewPublisher(publisherStub)
	if err != nil {
		t.Fatalf("NewPublisher() error = %v", err)
	}
	if err := publisher.Publish(context.Background(), port.IntegrationEvent{ID: "event-1", Topic: port.TopicMarkGenerated, SchemaVersion: "v1", OccurredAt: time.Now().UTC(), Payload: map[string]any{"markId": "mark-1"}}); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if len(publisherStub.messages) != 1 {
		t.Fatalf("published messages = %d", len(publisherStub.messages))
	}
}
