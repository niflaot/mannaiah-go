package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	ordersevent "mannaiah/module/orders/application/event"
	ordersdomain "mannaiah/module/orders/domain"
)

// UpdateStatus appends status values for order identifiers.
func (s *OrderService) UpdateStatus(ctx context.Context, id string, command UpdateStatusCommand) (*ordersdomain.Order, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, ErrInvalidID
	}
	entry := ordersdomain.StatusEntry{
		Status:      command.Status,
		Author:      strings.TrimSpace(command.Author),
		Description: strings.TrimSpace(command.Description),
		NoteOwner:   strings.TrimSpace(command.NoteOwner),
		Note:        strings.TrimSpace(command.Note),
		OccurredAt:  time.Now().UTC(),
	}
	if command.OccurredAt != nil && !command.OccurredAt.IsZero() {
		entry.OccurredAt = command.OccurredAt.UTC()
	}
	if strings.TrimSpace(entry.Author) == "" {
		return nil, ErrStatusAuthorRequired
	}
	if err := validateStatusEntry(entry); err != nil {
		return nil, err
	}
	current, err := s.repository.GetByID(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("load order for status update: %w", err)
	}
	if shouldIgnoreWooInboundMutation(command.Source, current.Realm) {
		s.enrichShippingWithBilling(ctx, current)
		return current, nil
	}

	entity, err := s.repository.AppendStatus(ctx, trimmedID, entry)
	if err != nil {
		return nil, fmt.Errorf("update order status: %w", err)
	}
	s.enrichShippingWithBilling(ctx, entity)
	if err := s.publisher.Publish(
		ctx,
		ordersevent.BuildOrderStatusUpdatedIntegrationEvent(*entity, command.Source),
	); err != nil {
		return nil, fmt.Errorf("publish order status updated event: %w", err)
	}

	return entity, nil
}
