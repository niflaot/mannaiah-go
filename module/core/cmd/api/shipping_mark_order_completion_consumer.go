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

// registerShippingMarkOrderCompletionConsumer registers mark-generated handlers that complete orders.
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

	return registrar.AddHandler(shippingport.TopicMarkGenerated, func(ctx context.Context, message coremsgbus.Message) error {
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
		if entity.CurrentStatus == ordersdomain.StatusCompleted {
			return nil
		}

		_, updateErr := ordersService.UpdateStatus(ctx, entity.ID, ordersapplication.UpdateStatusCommand{
			Status:      ordersdomain.StatusCompleted,
			Author:      "shipping",
			Description: "order completed after shipping mark generation",
			Source:      "shipping_mark_generated",
		})
		if updateErr != nil {
			if errors.Is(updateErr, ordersport.ErrNotFound) {
				return nil
			}
			if strings.Contains(strings.ToLower(updateErr.Error()), "order not found") {
				return nil
			}

			return fmt.Errorf("complete order from shipping mark generated: %w", updateErr)
		}

		return nil
	})
}
