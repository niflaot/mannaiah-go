package application

import (
	"context"
	"fmt"
	"strings"

	ordersevent "mannaiah/module/orders/application/event"
	ordersdomain "mannaiah/module/orders/domain"
)

// Update updates mutable order aggregate values by identifier.
func (s *OrderService) Update(ctx context.Context, id string, command UpdateCommand) (*ordersdomain.Order, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, ErrInvalidID
	}
	if command.Items == nil && command.ShippingAddress == nil && command.ShippingCharges == nil {
		return nil, ErrEmptyOrderUpdate
	}

	entity, err := s.repository.GetByID(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("load order for update: %w", err)
	}

	if command.Items != nil {
		items, resolveErr := s.resolveItems(ctx, *command.Items)
		if resolveErr != nil {
			return nil, resolveErr
		}
		entity.Items = items
	}
	if command.ShippingAddress != nil {
		entity.ShippingAddress = normalizeShippingCommand(*command.ShippingAddress)
		entity.HasCustomShippingAddress = true
	}
	if command.ShippingCharges != nil {
		entity.ShippingCharges = normalizeShippingCharges(*command.ShippingCharges)
	}

	entity.Normalize()
	if err := entity.Validate(); err != nil {
		return nil, err
	}
	if err := s.repository.Update(ctx, entity); err != nil {
		return nil, fmt.Errorf("update order: %w", err)
	}
	s.enrichShippingWithBilling(ctx, entity)
	if err := s.publisher.Publish(
		ctx,
		ordersevent.BuildOrderUpdatedIntegrationEvent(*entity, command.Source),
	); err != nil {
		return nil, fmt.Errorf("publish order updated event: %w", err)
	}

	return entity, nil
}
