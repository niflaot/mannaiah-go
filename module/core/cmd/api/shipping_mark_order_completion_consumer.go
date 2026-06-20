package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"
	coremsgbus "mannaiah/module/core/messaging/bus"
	ordersapplication "mannaiah/module/orders/application"
	ordersdomain "mannaiah/module/orders/domain"
	ordersport "mannaiah/module/orders/port"
	shippingport "mannaiah/module/shipping/port"
)

// shippingOrderStatusService defines order-status behavior required by shipping mark completion consumers.
type shippingOrderStatusService interface {
	// Get resolves order aggregate values by identifier.
	Get(ctx context.Context, id string) (*ordersdomain.Order, error)
	// UpdateStatus appends status values for order identifiers.
	UpdateStatus(ctx context.Context, id string, command ordersapplication.UpdateStatusCommand) (*ordersdomain.Order, error)
}

// registerShippingMarkOrderCompletionConsumer registers shipping-mark handlers that update order status.
func registerShippingMarkOrderCompletionConsumer(
	registrar coremsgbus.Registrar,
	ordersService shippingOrderStatusService,
	providedLogger *zap.Logger,
) error {
	if registrar == nil || ordersService == nil {
		return nil
	}
	logger := providedLogger
	if logger == nil {
		logger = zap.NewNop()
	}

	handleMarkGenerated := func(ctx context.Context, message coremsgbus.Message) error {
		payload, err := decodeShippingMarkGeneratedPayload(message)
		if err != nil {
			logger.Warn("decode shipping mark generated payload failed", zap.Error(err))
			return err
		}

		entity, getErr := ordersService.Get(ctx, payload.OrderID)
		if getErr != nil {
			if errors.Is(getErr, ordersport.ErrNotFound) {
				logger.Warn("shipping mark generated received for unknown order", zap.String("order_id", payload.OrderID), zap.String("mark_id", payload.MarkID))
				return nil
			}

			return fmt.Errorf("load order for shipping mark generated: %w", getErr)
		}
		if entity == nil {
			return nil
		}
		return nil
	}

	handleMarkVoided := func(ctx context.Context, message coremsgbus.Message) error {
		payload, err := decodeShippingMarkGeneratedPayload(message)
		if err != nil {
			logger.Warn("decode shipping mark voided payload failed", zap.Error(err))
			return err
		}

		entity, getErr := ordersService.Get(ctx, payload.OrderID)
		if getErr != nil {
			if errors.Is(getErr, ordersport.ErrNotFound) {
				logger.Warn("shipping mark voided received for unknown order", zap.String("order_id", payload.OrderID), zap.String("mark_id", payload.MarkID))
				return nil
			}

			return fmt.Errorf("load order for shipping mark voided: %w", getErr)
		}
		if entity == nil {
			return nil
		}

		voidedReference := firstNonEmpty(strings.TrimSpace(payload.TrackingNumber), strings.TrimSpace(payload.MarkID))
		description := "order returned to created after shipping mark voided"
		if voidedReference != "" {
			description += ": " + voidedReference
		}

		_, updateErr := ordersService.UpdateStatus(ctx, entity.ID, ordersapplication.UpdateStatusCommand{
			Status:      ordersdomain.StatusCreated,
			Author:      "shipping",
			Description: description,
			Source:      "shipping_mark_voided",
		})
		if updateErr != nil {
			if errors.Is(updateErr, ordersport.ErrNotFound) {
				return nil
			}
			if strings.Contains(strings.ToLower(updateErr.Error()), "order not found") {
				return nil
			}

			return fmt.Errorf("set order back to created from shipping mark voided: %w", updateErr)
		}

		return nil
	}

	if err := registrar.AddHandler(shippingport.TopicMarkGenerated, handleMarkGenerated); err != nil {
		return err
	}

	return registrar.AddHandler(shippingport.TopicMarkVoided, handleMarkVoided)
}
