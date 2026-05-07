package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go.uber.org/zap"
	"mannaiah/module/core/messaging/bus"
	shippingport "mannaiah/module/shipping/port"
	shopifyshipping "mannaiah/module/shopify/application/shipping/service"
)

var (
	// ErrNilShippingEventHandler is returned when shipping handlers are nil.
	ErrNilShippingEventHandler = errors.New("shopify shipping event handler must not be nil")
)

// ShippingEventHandler defines Shopify shipping event behavior.
type ShippingEventHandler interface {
	// HandleMarkGenerated handles generated mark events.
	HandleMarkGenerated(ctx context.Context, payload shopifyshipping.MarkGeneratedPayload) error
	// HandleMarkVoided handles voided mark events.
	HandleMarkVoided(ctx context.Context, payload shopifyshipping.MarkVoidedPayload) error
}

// ShippingConsumer consumes Mannaiah shipping events for Shopify fulfillment write-back.
type ShippingConsumer struct {
	// handler defines shipping handling dependencies.
	handler ShippingEventHandler
	// logger defines structured logging dependencies.
	logger *zap.Logger
}

// NewShippingConsumer creates Shopify shipping consumers.
func NewShippingConsumer(handler ShippingEventHandler, providedLogger *zap.Logger) (*ShippingConsumer, error) {
	if handler == nil {
		return nil, ErrNilShippingEventHandler
	}
	if providedLogger == nil {
		providedLogger = zap.NewNop()
	}
	return &ShippingConsumer{handler: handler, logger: providedLogger}, nil
}

// Register registers Shopify shipping handlers.
func (c *ShippingConsumer) Register(registrar bus.Registrar) error {
	if registrar == nil {
		return ErrNilRegistrar
	}
	if err := registrar.AddHandler(shippingport.TopicMarkGenerated, func(ctx context.Context, message bus.Message) error {
		return c.handleGenerated(ctx, message)
	}); err != nil {
		return fmt.Errorf("register shopify mark-generated handler: %w", err)
	}
	if err := registrar.AddHandler(shippingport.TopicMarkVoided, func(ctx context.Context, message bus.Message) error {
		return c.handleVoided(ctx, message)
	}); err != nil {
		return fmt.Errorf("register shopify mark-voided handler: %w", err)
	}
	return nil
}

func (c *ShippingConsumer) handleGenerated(ctx context.Context, message bus.Message) error {
	var payload shopifyshipping.MarkGeneratedPayload
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		c.logger.Warn("decode shopify mark-generated event failed", zap.Error(err))
		return nil
	}
	return c.handler.HandleMarkGenerated(ctx, payload)
}

func (c *ShippingConsumer) handleVoided(ctx context.Context, message bus.Message) error {
	var payload shopifyshipping.MarkVoidedPayload
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		c.logger.Warn("decode shopify mark-voided event failed", zap.Error(err))
		return nil
	}
	return c.handler.HandleMarkVoided(ctx, payload)
}
