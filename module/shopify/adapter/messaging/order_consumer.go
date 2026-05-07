package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go.uber.org/zap"
	"mannaiah/module/core/messaging/bus"
	ordersport "mannaiah/module/orders/port"
)

var (
	// ErrNilOrderEventHandler is returned when order handlers are nil.
	ErrNilOrderEventHandler = errors.New("shopify order event handler must not be nil")
	// ErrNilRegistrar is returned when registrars are nil.
	ErrNilRegistrar = errors.New("shopify event registrar must not be nil")
)

// OrderEventHandler defines Mannaiah order event handling behavior.
type OrderEventHandler interface {
	// HandleOrderEvent handles one order event payload.
	HandleOrderEvent(ctx context.Context, payload ordersport.OrderEventPayload) error
}

// OrderConsumer consumes Mannaiah order events for Shopify-realm write-back.
type OrderConsumer struct {
	// handler defines event handling dependencies.
	handler OrderEventHandler
	// logger defines structured logging dependencies.
	logger *zap.Logger
}

// NewOrderConsumer creates Shopify order event consumers.
func NewOrderConsumer(handler OrderEventHandler, providedLogger *zap.Logger) (*OrderConsumer, error) {
	if handler == nil {
		return nil, ErrNilOrderEventHandler
	}
	if providedLogger == nil {
		providedLogger = zap.NewNop()
	}
	return &OrderConsumer{handler: handler, logger: providedLogger}, nil
}

// Register registers Shopify order event handlers.
func (c *OrderConsumer) Register(registrar bus.Registrar) error {
	if registrar == nil {
		return ErrNilRegistrar
	}
	for _, topic := range []string{ordersport.TopicOrderUpdated, ordersport.TopicOrderStatusUpdated} {
		topicValue := topic
		if err := registrar.AddHandler(topicValue, func(ctx context.Context, message bus.Message) error {
			return c.handleMessage(ctx, topicValue, message)
		}); err != nil {
			return fmt.Errorf("register shopify order handler %q: %w", topicValue, err)
		}
	}
	return nil
}

func (c *OrderConsumer) handleMessage(ctx context.Context, topic string, message bus.Message) error {
	var payload ordersport.OrderEventPayload
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		c.logger.Warn("decode shopify order event failed", zap.String("topic", topic), zap.Error(err))
		return nil
	}
	if payload.Source == "" && message.Metadata != nil {
		payload.Source = message.Metadata["source"]
	}
	if err := c.handler.HandleOrderEvent(ctx, payload); err != nil {
		c.logger.Warn("handle shopify order event failed", zap.String("topic", topic), zap.Error(err))
		return err
	}
	return nil
}
