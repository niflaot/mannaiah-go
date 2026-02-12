package event

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"mannaiah/module/assets/port"
	"mannaiah/module/core/messaging/bus"
)

var (
	// ErrNilPublisher is returned when a nil core bus publisher is provided.
	ErrNilPublisher = errors.New("assets integration publisher must not be nil")
)

// Publisher defines integration event publication over core bus publishers.
type Publisher struct {
	// publisher defines abstract integration message transport.
	publisher bus.Publisher
}

var (
	// _ ensures Publisher satisfies integration event publisher contracts.
	_ port.IntegrationEventPublisher = (*Publisher)(nil)
)

// NewPublisher creates an integration event publisher over core message bus publishers.
func NewPublisher(publisher bus.Publisher) (*Publisher, error) {
	if publisher == nil {
		return nil, ErrNilPublisher
	}

	return &Publisher{publisher: publisher}, nil
}

// Publish emits integration events to the configured message bus.
func (p *Publisher) Publish(ctx context.Context, event port.IntegrationEvent) error {
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("marshal integration event payload: %w", err)
	}

	metadata := map[string]string{
		bus.MetadataSchemaVersion: event.SchemaVersion,
		bus.MetadataProducedAt:    event.OccurredAt.UTC().Format(time.RFC3339),
	}
	if event.CorrelationID != "" {
		metadata[bus.MetadataCorrelationID] = event.CorrelationID
	}
	if event.CausationID != "" {
		metadata[bus.MetadataCausationID] = event.CausationID
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
