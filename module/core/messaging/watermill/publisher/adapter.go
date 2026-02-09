package publisher

import (
	"context"
	"errors"
	"fmt"
	"strings"

	watermill "github.com/ThreeDotsLabs/watermill"
	wmmsg "github.com/ThreeDotsLabs/watermill/message"
	wmmiddleware "github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"mannaiah/module/core/messaging/bus"
	correlationctx "mannaiah/module/core/messaging/watermill/internal/correlation"
)

var (
	// ErrNilPublisher is returned when a nil Watermill publisher is provided.
	ErrNilPublisher = errors.New("watermill publisher must not be nil")
	// ErrMessageIDRequired is returned when a message id is not provided.
	ErrMessageIDRequired = errors.New("message id is required")
	// ErrMessageTopicRequired is returned when a message topic is not provided.
	ErrMessageTopicRequired = errors.New("message topic is required")
)

// Adapter adapts Watermill publishers to bus.Publisher contracts.
type Adapter struct {
	// publisher is the wrapped Watermill publisher implementation.
	publisher wmmsg.Publisher
}

var (
	// _ ensures Adapter implements bus.Publisher.
	_ bus.Publisher = (*Adapter)(nil)
)

// NewAdapter creates a bus.Publisher adapter over a Watermill publisher.
func NewAdapter(publisher wmmsg.Publisher) (*Adapter, error) {
	if publisher == nil {
		return nil, ErrNilPublisher
	}

	return &Adapter{
		publisher: publisher,
	}, nil
}

// Publish publishes a bus envelope through the wrapped Watermill publisher.
func (a *Adapter) Publish(ctx context.Context, msg bus.Message) error {
	if strings.TrimSpace(msg.ID) == "" {
		return ErrMessageIDRequired
	}
	if strings.TrimSpace(msg.Topic) == "" {
		return ErrMessageTopicRequired
	}

	outgoing := wmmsg.NewMessage(msg.ID, append([]byte(nil), msg.Payload...))
	for key, value := range msg.Metadata {
		outgoing.Metadata.Set(key, value)
	}

	applyCorrelationMetadata(ctx, outgoing)
	applyEventMetadata(msg.ID, outgoing)

	if err := a.publisher.Publish(msg.Topic, outgoing); err != nil {
		return fmt.Errorf("publish topic %q: %w", msg.Topic, err)
	}

	return nil
}

// applyCorrelationMetadata propagates correlation values from context when absent.
func applyCorrelationMetadata(ctx context.Context, message *wmmsg.Message) {
	if message.Metadata.Get(bus.MetadataCorrelationID) != "" {
		return
	}

	if ctx == nil {
		wmmiddleware.SetCorrelationID(watermill.NewUUID(), message)
		return
	}

	correlationID, ok := correlationctx.FromContext(ctx)
	if ok && strings.TrimSpace(correlationID) != "" {
		wmmiddleware.SetCorrelationID(correlationID, message)
		return
	}

	wmmiddleware.SetCorrelationID(watermill.NewUUID(), message)
}

// applyEventMetadata ensures required event metadata keys are present.
func applyEventMetadata(messageID string, message *wmmsg.Message) {
	if message.Metadata.Get(bus.MetadataEventID) == "" {
		message.Metadata.Set(bus.MetadataEventID, messageID)
	}
}
