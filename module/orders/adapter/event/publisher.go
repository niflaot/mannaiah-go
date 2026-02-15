package event

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"mannaiah/module/core/messaging/bus"
	ordersport "mannaiah/module/orders/port"
)

var (
	// ErrNilPublisher is returned when a nil bus publisher is provided.
	ErrNilPublisher = errors.New("orders integration publisher must not be nil")
)

// Publisher defines integration event publication over core bus publishers.
type Publisher struct {
	// publisher defines abstract integration message transport.
	publisher bus.Publisher
}

var (
	// _ ensures Publisher satisfies integration event publisher contracts.
	_ ordersport.IntegrationEventPublisher = (*Publisher)(nil)
)

// NewPublisher creates an integration event publisher over core message bus publishers.
func NewPublisher(publisher bus.Publisher) (*Publisher, error) {
	if publisher == nil {
		return nil, ErrNilPublisher
	}

	return &Publisher{publisher: publisher}, nil
}

// Publish emits integration events to the configured message bus.
func (p *Publisher) Publish(ctx context.Context, integrationEvent ordersport.IntegrationEvent) error {
	payload, err := json.Marshal(integrationEvent.Payload)
	if err != nil {
		return fmt.Errorf("marshal integration event payload: %w", err)
	}

	metadata := map[string]string{
		bus.MetadataSchemaVersion: integrationEvent.SchemaVersion,
		bus.MetadataProducedAt:    integrationEvent.OccurredAt.UTC().Format(time.RFC3339),
	}
	if integrationEvent.CorrelationID != "" {
		metadata[bus.MetadataCorrelationID] = integrationEvent.CorrelationID
	}
	if integrationEvent.CausationID != "" {
		metadata[bus.MetadataCausationID] = integrationEvent.CausationID
	}
	for key, value := range integrationEvent.Metadata {
		metadata[key] = value
	}

	if err := p.publisher.Publish(ctx, bus.Message{
		ID:       integrationEvent.ID,
		Topic:    integrationEvent.Topic,
		Payload:  payload,
		Metadata: metadata,
	}); err != nil {
		return fmt.Errorf("publish integration event topic %q: %w", integrationEvent.Topic, err)
	}

	return nil
}
