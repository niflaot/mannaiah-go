package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	contactsapplication "mannaiah/module/contacts/application"
	"mannaiah/module/core/messaging/bus"

	"go.uber.org/zap"
)

var (
	// ErrNilContactEventHandler is returned when a nil contact event handler is provided.
	ErrNilContactEventHandler = errors.New("shopify contact event handler must not be nil")
)

// ContactEventHandler defines contact integration event handling behavior.
type ContactEventHandler interface {
	// HandleContactEvent handles contact integration event payload values.
	HandleContactEvent(ctx context.Context, payload contactsapplication.ContactEventPayload) error
}

// ContactConsumer defines contact integration event consumer behavior over the core bus abstraction.
type ContactConsumer struct {
	// handler defines contact integration event handling dependencies.
	handler ContactEventHandler
	// logger defines structured logging dependencies.
	logger *zap.Logger
}

// NewContactConsumer creates Shopify contact integration event consumers.
func NewContactConsumer(handler ContactEventHandler, providedLogger *zap.Logger) (*ContactConsumer, error) {
	if handler == nil {
		return nil, ErrNilContactEventHandler
	}

	logger := providedLogger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &ContactConsumer{handler: handler, logger: logger}, nil
}

// Register registers contact integration event handlers on the provided registrar.
func (c *ContactConsumer) Register(registrar bus.Registrar) error {
	if registrar == nil {
		return ErrNilRegistrar
	}

	for _, topic := range []string{contactsapplication.TopicContactCreated, contactsapplication.TopicContactUpdated} {
		topicValue := topic
		c.logger.Info("register shopify contact integration handler", zap.String("topic", topicValue))
		if err := registrar.AddHandler(topicValue, func(ctx context.Context, message bus.Message) error {
			return c.handleMessage(ctx, topicValue, message)
		}); err != nil {
			return fmt.Errorf("register topic handler %q: %w", topicValue, err)
		}
	}

	return nil
}

func (c *ContactConsumer) handleMessage(ctx context.Context, topic string, message bus.Message) error {
	var payload contactsapplication.ContactEventPayload
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		c.logger.Warn("decode shopify contact integration event failed", zap.String("topic", topic), zap.Error(err))
		return nil
	}
	c.logger.Info(
		"shopify contact integration event received",
		zap.String("topic", topic),
		zap.String("message_id", message.ID),
		zap.String("contact_id", payload.ID),
		zap.Bool("has_email", payload.Email != ""),
		zap.Bool("has_shopify_customer_metadata", payload.Metadata["shopify_customer_id"] != ""),
	)

	if err := c.handler.HandleContactEvent(ctx, payload); err != nil {
		if isTemporaryShopifyUnavailable(err) {
			c.logger.Warn("defer shopify contact integration event: shopify is temporarily unavailable", zap.String("topic", topic), zap.String("contact_id", payload.ID), zap.Error(err))
			return nil
		}
		c.logger.Warn("handle shopify contact integration event failed", zap.String("topic", topic), zap.Error(err))
		return err
	}
	c.logger.Info("shopify contact integration event handled", zap.String("topic", topic), zap.String("contact_id", payload.ID))

	return nil
}

func isTemporaryShopifyUnavailable(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "shopify integration is unavailable")
}
