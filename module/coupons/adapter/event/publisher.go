// Package event defines the coupon integration event publisher adapter.
package event

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"mannaiah/module/core/messaging/bus"
	"mannaiah/module/coupons/port"
)

// Publisher defines coupon integration event publication adapter behavior.
type Publisher struct {
	// inner defines the underlying message bus publisher.
	inner bus.Publisher
}

// NewPublisher creates coupon integration event publishers.
func NewPublisher(inner bus.Publisher) (*Publisher, error) {
	if inner == nil {
		return nil, errors.New("coupon event publisher: inner bus publisher must not be nil")
	}
	return &Publisher{inner: inner}, nil
}

// Publish serializes and emits a coupon integration event to the message bus.
func (p *Publisher) Publish(ctx context.Context, event port.IntegrationEvent) error {
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("marshal coupon event payload: %w", err)
	}

	metadata := make(map[string]string)
	for k, v := range event.Metadata {
		metadata[k] = v
	}
	metadata[bus.MetadataSchemaVersion] = event.SchemaVersion
	metadata[bus.MetadataProducedAt] = event.OccurredAt.Format(time.RFC3339)
	metadata[bus.MetadataEventID] = event.ID

	msg := bus.Message{
		ID:       event.ID,
		Topic:    event.Topic,
		Payload:  payload,
		Metadata: metadata,
	}

	if err := p.inner.Publish(ctx, msg); err != nil {
		return fmt.Errorf("publish coupon event %q: %w", event.Topic, err)
	}

	return nil
}
