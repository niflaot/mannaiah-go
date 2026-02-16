package application

import (
	"context"
	"fmt"
	"math"
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
	if shouldIgnoreWooInboundMutation(command.Source, entity.Realm) {
		s.enrichShippingWithBilling(ctx, entity)
		return entity, nil
	}
	previous := snapshotMutableState(*entity)

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
	if !hasMutableStateChanges(previous, snapshotMutableState(*entity)) {
		s.enrichShippingWithBilling(ctx, entity)
		return entity, nil
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

// mutableStateSnapshot defines mutable order-state snapshot values.
type mutableStateSnapshot struct {
	// Items defines item state rows.
	Items []ordersdomain.Item
	// ShippingAddress defines shipping-address state values.
	ShippingAddress ordersdomain.ShippingAddress
	// HasCustomShippingAddress reports custom shipping-state values.
	HasCustomShippingAddress bool
	// ShippingCharges defines shipping-charge state values.
	ShippingCharges []ordersdomain.ShippingCharge
}

// snapshotMutableState snapshots mutable order state values.
func snapshotMutableState(value ordersdomain.Order) mutableStateSnapshot {
	return mutableStateSnapshot{
		Items:                    append([]ordersdomain.Item{}, value.Items...),
		ShippingAddress:          value.ShippingAddress,
		HasCustomShippingAddress: value.HasCustomShippingAddress,
		ShippingCharges:          append([]ordersdomain.ShippingCharge{}, value.ShippingCharges...),
	}
}

// hasMutableStateChanges reports whether mutable order state changed.
func hasMutableStateChanges(left mutableStateSnapshot, right mutableStateSnapshot) bool {
	if left.HasCustomShippingAddress != right.HasCustomShippingAddress {
		return true
	}
	if !shippingEqual(left.ShippingAddress, right.ShippingAddress) {
		return true
	}
	if len(left.Items) != len(right.Items) {
		return true
	}
	for index := range left.Items {
		if !itemsEqual(left.Items[index], right.Items[index]) {
			return true
		}
	}
	if len(left.ShippingCharges) != len(right.ShippingCharges) {
		return true
	}
	for index := range left.ShippingCharges {
		if !shippingChargesEqual(left.ShippingCharges[index], right.ShippingCharges[index]) {
			return true
		}
	}

	return false
}

// itemsEqual reports whether item values are equivalent.
func itemsEqual(left ordersdomain.Item, right ordersdomain.Item) bool {
	return strings.EqualFold(strings.TrimSpace(left.SKU), strings.TrimSpace(right.SKU)) &&
		strings.EqualFold(strings.TrimSpace(left.AlternateName), strings.TrimSpace(right.AlternateName)) &&
		left.Quantity == right.Quantity &&
		math.Abs(left.Value-right.Value) <= 0.000001 &&
		strings.EqualFold(strings.TrimSpace(left.ProductID), strings.TrimSpace(right.ProductID)) &&
		strings.EqualFold(strings.TrimSpace(string(left.ResolutionSource)), strings.TrimSpace(string(right.ResolutionSource)))
}

// shippingChargesEqual reports whether shipping charge values are equivalent.
func shippingChargesEqual(left ordersdomain.ShippingCharge, right ordersdomain.ShippingCharge) bool {
	return strings.EqualFold(strings.TrimSpace(left.MethodID), strings.TrimSpace(right.MethodID)) &&
		strings.EqualFold(strings.TrimSpace(left.MethodTitle), strings.TrimSpace(right.MethodTitle)) &&
		math.Abs(left.Price-right.Price) <= 0.000001
}
