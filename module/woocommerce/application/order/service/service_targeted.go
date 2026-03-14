package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	wooorderevent "mannaiah/module/woocommerce/application/order/event"
	"mannaiah/module/woocommerce/port"
)

// commandResolver defines command-resolution behavior for sync flows.
type commandResolver func(ctx context.Context, summary *SyncSummary) ([]port.OrderSyncCommand, error)

// singleOrderSource defines optional source behavior for direct Woo order retrieval.
type singleOrderSource interface {
	// GetOrderByID resolves one Woo order by identifier.
	GetOrderByID(ctx context.Context, orderID int) (order port.WooOrder, err error)
}

// syncOrdersWithResolver performs sync lifecycle behavior using provided command-resolution behavior.
func (s *OrderSyncService) syncOrdersWithResolver(ctx context.Context, trigger string, resolver commandResolver) (summary *SyncSummary, err error) {
	summary = &SyncSummary{Trigger: normalizeTrigger(trigger)}
	runID := s.startSyncRunRecord(ctx, summary.Trigger)
	defer func() {
		s.finishSyncRunRecord(ctx, runID, summary, err)
	}()

	s.publishEvent(ctx, wooorderevent.NewSyncStartedEvent(summary.Trigger))

	if err := s.ValidateIntegration(ctx); err != nil {
		s.publishEvent(ctx, wooorderevent.NewSyncFailedEvent(toEventSummary(*summary), err))
		return nil, err
	}

	pendingCommands, err := resolver(ctx, summary)
	if err != nil {
		s.publishEvent(ctx, wooorderevent.NewSyncFailedEvent(toEventSummary(*summary), err))
		return nil, err
	}

	if err := s.processCommands(ctx, pendingCommands, summary); err != nil {
		wrappedErr := fmt.Errorf("process woocommerce orders sync (%s): %w", formatSyncProgress(summary), err)
		s.publishEvent(ctx, wooorderevent.NewSyncFailedEvent(toEventSummary(*summary), wrappedErr))
		return nil, wrappedErr
	}

	s.publishEvent(ctx, wooorderevent.NewSyncCompletedEvent(toEventSummary(*summary)))
	return summary, nil
}

// resolveAllCommands resolves sync commands from paginated Woo order listing behavior.
func (s *OrderSyncService) resolveAllCommands(ctx context.Context, summary *SyncSummary) ([]port.OrderSyncCommand, error) {
	commandIndexByIdentifier := map[string]int{}
	pendingCommands := make([]port.OrderSyncCommand, 0)
	page := 1
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		orders, hasNext, err := s.loadPage(ctx, page)
		if err != nil {
			return nil, fmt.Errorf("list woocommerce orders page %d (%s): %w", page, formatSyncProgress(summary), err)
		}
		if len(orders) == 0 {
			break
		}

		pendingCommands = collectCommandsFromOrders(orders, commandIndexByIdentifier, pendingCommands, summary, s.logger)

		if !hasNext {
			break
		}
		page++
	}

	return pendingCommands, nil
}

// resolveCommandsByOrderID resolves sync commands for one Woo order identifier.
func (s *OrderSyncService) resolveCommandsByOrderID(ctx context.Context, summary *SyncSummary, orderID int) ([]port.OrderSyncCommand, error) {
	if source, ok := s.source.(singleOrderSource); ok {
		command, found, err := s.resolveOneCommandBySourceID(ctx, source, orderID, summary)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, ErrOrderNotFound
		}

		return command, nil
	}

	command, found, err := s.resolveOneCommandByListing(ctx, orderID, summary)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrOrderNotFound
	}

	return command, nil
}

// resolveOneCommandBySourceID resolves one sync command using direct source lookup behavior.
func (s *OrderSyncService) resolveOneCommandBySourceID(
	ctx context.Context,
	source singleOrderSource,
	orderID int,
	summary *SyncSummary,
) ([]port.OrderSyncCommand, bool, error) {
	var order port.WooOrder
	err := s.executeWithBreaker(s.sourceBreaker, ErrIntegrationUnavailable, func() error {
		var lookupErr error
		order, lookupErr = source.GetOrderByID(ctx, orderID)
		return lookupErr
	})
	if err != nil {
		if errors.Is(err, ErrIntegrationUnavailable) {
			return nil, false, err
		}
		if isSourceNotFoundError(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("get woocommerce order %d: %w", orderID, err)
	}

	pendingCommands := collectCommandsFromOrders(
		[]port.WooOrder{order},
		map[string]int{},
		make([]port.OrderSyncCommand, 0, 1),
		summary,
		s.logger,
	)

	return pendingCommands, true, nil
}

// resolveOneCommandByListing resolves one sync command by scanning paginated order listing behavior.
func (s *OrderSyncService) resolveOneCommandByListing(ctx context.Context, orderID int, summary *SyncSummary) ([]port.OrderSyncCommand, bool, error) {
	page := 1
	for {
		if err := ctx.Err(); err != nil {
			return nil, false, err
		}

		orders, hasNext, err := s.loadPage(ctx, page)
		if err != nil {
			return nil, false, fmt.Errorf("list woocommerce orders page %d (%s): %w", page, formatSyncProgress(summary), err)
		}
		if len(orders) == 0 {
			return nil, false, nil
		}

		for _, order := range orders {
			if order.ID != orderID {
				continue
			}

			pendingCommands := collectCommandsFromOrders(
				[]port.WooOrder{order},
				map[string]int{},
				make([]port.OrderSyncCommand, 0, 1),
				summary,
				s.logger,
			)
			return pendingCommands, true, nil
		}

		if !hasNext {
			return nil, false, nil
		}
		page++
	}
}

// isSourceNotFoundError reports whether source errors represent missing Woo order identifiers.
func isSourceNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	value := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(value, "404") || strings.Contains(value, "not found")
}
