package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	ordersevent "mannaiah/module/orders/application/event"
	ordersdomain "mannaiah/module/orders/domain"
)

// AddComment appends comment values for order identifiers.
func (s *OrderService) AddComment(ctx context.Context, id string, command AddCommentCommand) (*ordersdomain.Order, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, ErrInvalidID
	}
	comment := ordersdomain.Comment{
		Author:     strings.TrimSpace(command.Author),
		Comment:    strings.TrimSpace(command.Comment),
		Internal:   command.Internal,
		OccurredAt: time.Now().UTC(),
	}
	if command.OccurredAt != nil && !command.OccurredAt.IsZero() {
		comment.OccurredAt = command.OccurredAt.UTC()
	}
	if err := validateComment(comment); err != nil {
		return nil, err
	}
	current, err := s.repository.GetByID(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("load order for comment append: %w", err)
	}
	if shouldIgnoreWooInboundMutation(command.Source, current.Realm) {
		s.enrichShippingWithBilling(ctx, current)
		return current, nil
	}

	updated, err := s.repository.AppendComment(ctx, trimmedID, comment)
	if err != nil {
		return nil, fmt.Errorf("add order comment: %w", err)
	}
	s.enrichShippingWithBilling(ctx, updated)
	if err := s.publisher.Publish(
		ctx,
		ordersevent.BuildOrderUpdatedIntegrationEvent(*updated, command.Source),
	); err != nil {
		return nil, fmt.Errorf("publish order updated event: %w", err)
	}

	return updated, nil
}

// validateComment validates order comment payload invariants.
func validateComment(comment ordersdomain.Comment) error {
	if strings.TrimSpace(comment.Author) == "" {
		return ordersdomain.ErrCommentAuthorRequired
	}
	if strings.TrimSpace(comment.Comment) == "" {
		return ordersdomain.ErrCommentTextRequired
	}

	return nil
}
