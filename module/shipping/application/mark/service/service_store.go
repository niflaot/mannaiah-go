package service

import (
	"context"
	"strings"

	"mannaiah/module/shipping/domain"
)

// ListByOrderID lists all marks for one order identifier.
func (s *Service) ListByOrderID(ctx context.Context, orderID string) ([]domain.ShippingMark, error) {
	if s == nil || s.repository == nil {
		return []domain.ShippingMark{}, nil
	}

	return s.repository.ListByOrderID(ctx, strings.TrimSpace(orderID))
}
