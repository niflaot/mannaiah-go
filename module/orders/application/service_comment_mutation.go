package application

import (
	"context"
	"fmt"
	"strings"

	ordersevent "mannaiah/module/orders/application/event"
	ordersdomain "mannaiah/module/orders/domain"
	ordersport "mannaiah/module/orders/port"
)

// UpdateComment updates comment values for order identifiers.
func (s *OrderService) UpdateComment(ctx context.Context, id string, commentID string, command UpdateCommentCommand) (*ordersdomain.Order, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, ErrInvalidID
	}
	trimmedCommentID := strings.TrimSpace(commentID)
	if trimmedCommentID == "" {
		return nil, ErrInvalidCommentID
	}
	if command.Author == nil && command.Comment == nil && command.Internal == nil {
		return nil, ErrEmptyCommentUpdate
	}

	current, err := s.repository.GetByID(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("load order for comment update: %w", err)
	}
	if shouldIgnoreWooInboundMutation(command.Source, current.Realm) {
		s.enrichShippingWithBilling(ctx, current)
		return current, nil
	}

	nextComment, err := mergeCommentUpdate(current.Comments, trimmedCommentID, command)
	if err != nil {
		return nil, err
	}
	if err := validateComment(nextComment); err != nil {
		return nil, err
	}

	updated, err := s.repository.UpdateComment(ctx, trimmedID, trimmedCommentID, nextComment)
	if err != nil {
		return nil, fmt.Errorf("update order comment: %w", err)
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

// DeleteComment deletes comment values for order identifiers.
func (s *OrderService) DeleteComment(ctx context.Context, id string, commentID string, command DeleteCommentCommand) (*ordersdomain.Order, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, ErrInvalidID
	}
	trimmedCommentID := strings.TrimSpace(commentID)
	if trimmedCommentID == "" {
		return nil, ErrInvalidCommentID
	}

	current, err := s.repository.GetByID(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("load order for comment delete: %w", err)
	}
	if shouldIgnoreWooInboundMutation(command.Source, current.Realm) {
		s.enrichShippingWithBilling(ctx, current)
		return current, nil
	}

	updated, err := s.repository.DeleteComment(ctx, trimmedID, trimmedCommentID)
	if err != nil {
		return nil, fmt.Errorf("delete order comment: %w", err)
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

// mergeCommentUpdate merges existing comment values with update command values.
func mergeCommentUpdate(comments []ordersdomain.Comment, commentID string, command UpdateCommentCommand) (ordersdomain.Comment, error) {
	trimmedCommentID := strings.TrimSpace(commentID)
	for _, value := range comments {
		if strings.TrimSpace(value.ID) != trimmedCommentID {
			continue
		}

		next := value
		if command.Author != nil {
			next.Author = strings.TrimSpace(*command.Author)
		}
		if command.Comment != nil {
			next.Comment = strings.TrimSpace(*command.Comment)
		}
		if command.Internal != nil {
			next.Internal = *command.Internal
		}

		return next, nil
	}

	return ordersdomain.Comment{}, ordersport.ErrCommentNotFound
}
