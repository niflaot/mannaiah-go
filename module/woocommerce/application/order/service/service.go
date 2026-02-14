package service

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"
	wooorderevent "mannaiah/module/woocommerce/application/order/event"
	"mannaiah/module/woocommerce/port"
)

var (
	// ErrNilSource is returned when a nil WooCommerce order source is used.
	ErrNilSource = errors.New("woocommerce order source must not be nil")
	// ErrNilTarget is returned when a nil order sync target is used.
	ErrNilTarget = errors.New("woocommerce order sync target must not be nil")
	// ErrSyncDisabled is returned when order sync is disabled by configuration.
	ErrSyncDisabled = errors.New("woocommerce orders sync is disabled")
	// ErrIntegrationUnavailable is returned when WooCommerce integration is unavailable.
	ErrIntegrationUnavailable = errors.New("woocommerce integration is unavailable")
	// ErrUpsertUnavailable is returned when order-upsert dependencies are unavailable.
	ErrUpsertUnavailable = errors.New("woocommerce order upsert dependency is unavailable")
)

// SyncConfig defines sync behavior configuration values.
type SyncConfig struct {
	// Enabled defines whether order sync behavior is enabled.
	Enabled bool
	// PageSize defines WooCommerce order page sizes.
	PageSize int
	// WorkerCount defines concurrent order upsert workers.
	WorkerCount int
}

// CircuitBreaker defines circuit-breaker behavior used by sync dependencies.
type CircuitBreaker interface {
	// Execute runs operations behind a circuit breaker.
	Execute(operation func() error) error
	// IsOpenError reports whether errors represent open-circuit rejections.
	IsOpenError(err error) bool
}

// CircuitBreakers defines optional per-dependency circuit-breaker wiring.
type CircuitBreakers struct {
	// Source guards WooCommerce API calls.
	Source CircuitBreaker
	// Upsert guards order-upsert calls.
	Upsert CircuitBreaker
}

// SyncSummary defines order sync execution results.
type SyncSummary struct {
	// Trigger defines sync trigger names.
	Trigger string `json:"trigger"`
	// Processed defines upsert-attempt counts.
	Processed int `json:"processed"`
	// Created defines created order counts.
	Created int `json:"created"`
	// Updated defines updated order counts.
	Updated int `json:"updated"`
	// Unchanged defines no-op order counts.
	Unchanged int `json:"unchanged"`
	// Skipped defines skipped-order counts.
	Skipped int `json:"skipped"`
	// Failed defines failed upsert counts.
	Failed int `json:"failed"`
}

// Service defines WooCommerce order sync use-case behavior.
type Service interface {
	// ValidateIntegration verifies sync preconditions and WooCommerce connectivity.
	ValidateIntegration(ctx context.Context) error
	// SyncOrders performs order synchronization and emits integration events.
	SyncOrders(ctx context.Context, trigger string) (*SyncSummary, error)
}

// OrderSyncService defines WooCommerce order sync use-case dependencies.
type OrderSyncService struct {
	// source defines WooCommerce order retrieval dependencies.
	source port.OrderSource
	// target defines order upsert dependencies.
	target port.OrderSyncTarget
	// publisher defines integration event publication dependencies.
	publisher port.IntegrationEventPublisher
	// logger defines structured log dependencies.
	logger *zap.Logger
	// cfg defines sync behavior configuration values.
	cfg SyncConfig
	// sourceBreaker guards WooCommerce API calls.
	sourceBreaker CircuitBreaker
	// upsertBreaker guards order-upsert calls.
	upsertBreaker CircuitBreaker
}

// upsertResult defines command upsert result payload values.
type upsertResult struct {
	// outcome defines upsert outcomes.
	outcome port.UpsertOutcome
	// err defines upsert execution errors.
	err error
}

var (
	// _ ensures OrderSyncService satisfies service contracts.
	_ Service = (*OrderSyncService)(nil)
)

// NewService creates WooCommerce order sync services.
func NewService(cfg SyncConfig, source port.OrderSource, target port.OrderSyncTarget, publisher port.IntegrationEventPublisher, providedLogger *zap.Logger, breakers ...CircuitBreakers) (*OrderSyncService, error) {
	if source == nil {
		return nil, ErrNilSource
	}
	if target == nil {
		return nil, ErrNilTarget
	}

	resolvedBreakers := resolveCircuitBreakers(breakers)

	return &OrderSyncService{
		source:        source,
		target:        target,
		publisher:     wooorderevent.ResolvePublisher(publisher),
		logger:        resolveLogger(providedLogger),
		cfg:           normalizeSyncConfig(cfg),
		sourceBreaker: resolvedBreakers.Source,
		upsertBreaker: resolvedBreakers.Upsert,
	}, nil
}

// ValidateIntegration verifies sync preconditions and WooCommerce connectivity.
func (s *OrderSyncService) ValidateIntegration(ctx context.Context) error {
	if !s.cfg.Enabled {
		return ErrSyncDisabled
	}

	err := s.executeWithBreaker(s.sourceBreaker, ErrIntegrationUnavailable, func() error {
		return s.source.Validate(ctx)
	})
	if err != nil {
		if errors.Is(err, ErrIntegrationUnavailable) {
			return err
		}
		return fmt.Errorf("%w: %v", ErrIntegrationUnavailable, err)
	}

	return nil
}

// SyncOrders performs order synchronization and emits integration events.
func (s *OrderSyncService) SyncOrders(ctx context.Context, trigger string) (*SyncSummary, error) {
	summary := &SyncSummary{Trigger: normalizeTrigger(trigger)}
	s.publishEvent(ctx, wooorderevent.NewSyncStartedEvent(summary.Trigger))

	if err := s.ValidateIntegration(ctx); err != nil {
		s.publishEvent(ctx, wooorderevent.NewSyncFailedEvent(toEventSummary(*summary), err))
		return nil, err
	}

	commandIndexByIdentifier := map[string]int{}
	pendingCommands := make([]port.OrderSyncCommand, 0)
	page := 1
	for {
		if err := ctx.Err(); err != nil {
			s.publishEvent(ctx, wooorderevent.NewSyncFailedEvent(toEventSummary(*summary), err))
			return nil, err
		}

		orders, hasNext, err := s.loadPage(ctx, page)
		if err != nil {
			wrappedErr := fmt.Errorf("list woocommerce orders page %d (%s): %w", page, formatSyncProgress(summary), err)
			s.publishEvent(ctx, wooorderevent.NewSyncFailedEvent(toEventSummary(*summary), wrappedErr))
			return nil, wrappedErr
		}

		if len(orders) == 0 {
			break
		}

		pendingCommands = collectCommandsFromOrders(orders, commandIndexByIdentifier, pendingCommands, summary)

		if !hasNext {
			break
		}
		page++
	}

	if err := s.processCommands(ctx, pendingCommands, summary); err != nil {
		wrappedErr := fmt.Errorf("process woocommerce orders sync (%s): %w", formatSyncProgress(summary), err)
		s.publishEvent(ctx, wooorderevent.NewSyncFailedEvent(toEventSummary(*summary), wrappedErr))
		return nil, wrappedErr
	}

	s.publishEvent(ctx, wooorderevent.NewSyncCompletedEvent(toEventSummary(*summary)))
	return summary, nil
}

// loadPage retrieves one WooCommerce order page with breaker protection.
func (s *OrderSyncService) loadPage(ctx context.Context, page int) (orders []port.WooOrder, hasNext bool, err error) {
	err = s.executeWithBreaker(s.sourceBreaker, ErrIntegrationUnavailable, func() error {
		var listErr error
		orders, hasNext, listErr = s.source.ListOrders(ctx, page, s.cfg.PageSize)
		return listErr
	})
	if err != nil {
		return nil, false, err
	}

	return orders, hasNext, nil
}

// formatSyncProgress formats sync summary counters for error diagnostics.
func formatSyncProgress(summary *SyncSummary) string {
	if summary == nil {
		return "trigger=unknown processed=0 created=0 updated=0 unchanged=0 skipped=0 failed=0"
	}

	return fmt.Sprintf(
		"trigger=%s processed=%d created=%d updated=%d unchanged=%d skipped=%d failed=%d",
		summary.Trigger,
		summary.Processed,
		summary.Created,
		summary.Updated,
		summary.Unchanged,
		summary.Skipped,
		summary.Failed,
	)
}
