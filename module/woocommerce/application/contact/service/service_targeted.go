package service

import (
	"context"
	"strings"

	"mannaiah/module/woocommerce/port"
)

// pageLoader defines page-fetch behavior for contact sync flows.
type pageLoader func(ctx context.Context, page int) (orders []port.WooOrder, hasNext bool, err error)

// searchableOrderSource defines optional source behavior for text-based Woo order searches.
type searchableOrderSource interface {
	// SearchOrders retrieves paginated order values filtered by search terms.
	SearchOrders(ctx context.Context, search string, page int, pageSize int) (orders []port.WooOrder, hasNext bool, err error)
}

// normalizeEmailKey normalizes email values used for targeted sync behavior.
func normalizeEmailKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// newEmailPageLoader resolves page-loader behavior for targeted email sync behavior.
func (s *ContactSyncService) newEmailPageLoader(email string) pageLoader {
	return func(ctx context.Context, page int) (orders []port.WooOrder, hasNext bool, err error) {
		if source, ok := s.source.(searchableOrderSource); ok {
			return s.loadPageBySearch(ctx, source, email, page)
		}

		orders, hasNext, err = s.loadPage(ctx, page)
		if err != nil {
			return nil, false, err
		}

		return filterOrdersByEmail(orders, email), hasNext, nil
	}
}

// loadPageBySearch loads one WooCommerce order page using search behavior.
func (s *ContactSyncService) loadPageBySearch(
	ctx context.Context,
	source searchableOrderSource,
	search string,
	page int,
) (orders []port.WooOrder, hasNext bool, err error) {
	err = s.executeWithBreaker(s.sourceBreaker, ErrIntegrationUnavailable, func() error {
		var listErr error
		orders, hasNext, listErr = source.SearchOrders(ctx, search, page, s.cfg.PageSize)
		return listErr
	})
	if err != nil {
		return nil, false, err
	}

	return filterOrdersByEmail(orders, search), hasNext, nil
}

// filterOrdersByEmail filters order values by billing email values.
func filterOrdersByEmail(orders []port.WooOrder, email string) []port.WooOrder {
	resolvedEmail := normalizeEmailKey(email)
	if resolvedEmail == "" || len(orders) == 0 {
		return nil
	}

	filtered := make([]port.WooOrder, 0, len(orders))
	for _, order := range orders {
		if normalizeEmailKey(order.BillingEmail) != resolvedEmail {
			continue
		}
		filtered = append(filtered, order)
	}

	return filtered
}
