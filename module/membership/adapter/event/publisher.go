package event

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"mannaiah/module/core/messaging/bus"
	"mannaiah/module/membership/port"
)

var (
	// ErrNilPublisher is returned when nil bus publisher dependencies are provided.
	ErrNilPublisher = errors.New("membership integration publisher must not be nil")
)

// Publisher defines integration event publication over core bus publishers.
type Publisher struct {
	// publisher defines abstract message transport.
	publisher bus.Publisher
}

var (
	// _ ensures Publisher satisfies integration event publisher contracts.
	_ port.IntegrationEventPublisher = (*Publisher)(nil)
)

// NewPublisher creates membership integration event publishers.
func NewPublisher(publisher bus.Publisher) (*Publisher, error) {
	if publisher == nil {
		return nil, ErrNilPublisher
	}

	return &Publisher{publisher: publisher}, nil
}

// Publish emits integration events to the configured message transport.
func (p *Publisher) Publish(ctx context.Context, event port.IntegrationEvent) error {
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("marshal integration event payload: %w", err)
	}

	metadata := map[string]string{
		bus.MetadataSchemaVersion: event.SchemaVersion,
		bus.MetadataProducedAt:    event.OccurredAt.UTC().Format(time.RFC3339),
	}
	for key, value := range event.Metadata {
		metadata[key] = value
	}

	if err := p.publisher.Publish(ctx, bus.Message{
		ID:       event.ID,
		Topic:    event.Topic,
		Payload:  payload,
		Metadata: metadata,
	}); err != nil {
		return fmt.Errorf("publish integration event topic %q: %w", event.Topic, err)
	}

	return nil
}
