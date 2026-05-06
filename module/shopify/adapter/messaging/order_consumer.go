package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"mannaiah/module/core/messaging/bus"
	ordersport "mannaiah/module/orders/port"

	"go.uber.org/zap"
)

var (
	// ErrNilOrderEventHandler is returned when a nil order event handler is provided.
	ErrNilOrderEventHandler = errors.New("shopify order event handler must not be nil")
	// ErrNilRegistrar is returned when a nil event registrar is provided.
	ErrNilRegistrar = errors.New("shopify event registrar must not be nil")
)

// OrderEventHandler defines order integration event handling behavior.
type OrderEventHandler interface {
	// HandleOrderEvent handles order integration event payload values.
	HandleOrderEvent(ctx context.Context, payload ordersport.OrderEventPayload) error
}

// OrderConsumer defines order integration event consumer behavior over the core bus abstraction.
type OrderConsumer struct {
	// handler defines order integration event handling dependencies.
	handler OrderEventHandler
	// logger defines structured logging dependencies.
	logger *zap.Logger
}

// NewOrderConsumer creates Shopify order integration event consumers.
func NewOrderConsumer(handler OrderEventHandler, providedLogger *zap.Logger) (*OrderConsumer, error) {
	if handler == nil {
		return nil, ErrNilOrderEventHandler
	}

	logger := providedLogger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &OrderConsumer{handler: handler, logger: logger}, nil
}

// Register registers order integration event handlers on the provided registrar.
func (c *OrderConsumer) Register(registrar bus.Registrar) error {
	if registrar == nil {
		return ErrNilRegistrar
	}

	for _, topic := range []string{ordersport.TopicOrderCreated, ordersport.TopicOrderUpdated, ordersport.TopicOrderStatusUpdated} {
		topicValue := topic
		if err := registrar.AddHandler(topicValue, func(ctx context.Context, message bus.Message) error {
			return c.handleMessage(ctx, topicValue, message)
		}); err != nil {
			return fmt.Errorf("register topic handler %q: %w", topicValue, err)
		}
	}

	return nil
}

func (c *OrderConsumer) handleMessage(ctx context.Context, topic string, message bus.Message) error {
	var payload ordersport.OrderEventPayload
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		c.logger.Warn("decode shopify order integration event failed", zap.String("topic", topic), zap.Error(err))
		return nil
	}
	if payload.Source == "" && message.Metadata != nil {
		payload.Source = message.Metadata["source"]
	}

	if err := c.handler.HandleOrderEvent(ctx, payload); err != nil {
		c.logger.Warn("handle shopify order integration event failed", zap.String("topic", topic), zap.Error(err))
		return err
	}

	return nil
}
