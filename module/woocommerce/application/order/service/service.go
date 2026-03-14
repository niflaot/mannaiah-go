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
	// ErrInvalidOrderID is returned when targeted order-sync identifier values are invalid.
	ErrInvalidOrderID = errors.New("woocommerce order id must be greater than zero")
	// ErrOrderNotFound is returned when targeted order-sync identifiers are not present in Woo orders.
	ErrOrderNotFound = errors.New("woocommerce order not found")
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
	// SyncOrderByID performs targeted order synchronization for one Woo order identifier.
	SyncOrderByID(ctx context.Context, trigger string, orderID int) (*SyncSummary, error)
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
	// syncRecorder defines optional sync-run recording dependencies.
	syncRecorder port.SyncRecorder
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
		syncRecorder:  port.NoopSyncRecorder{},
	}, nil
}

// SetSyncRecorder configures optional sync run recording dependencies.
func (s *OrderSyncService) SetSyncRecorder(recorder port.SyncRecorder) {
	if s == nil {
		return
	}
	if recorder == nil {
		s.syncRecorder = port.NoopSyncRecorder{}
		return
	}

	s.syncRecorder = recorder
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
	return s.syncOrdersWithResolver(ctx, trigger, s.resolveAllCommands)
}

// SyncOrderByID performs targeted order synchronization for one Woo order identifier.
func (s *OrderSyncService) SyncOrderByID(ctx context.Context, trigger string, orderID int) (*SyncSummary, error) {
	if orderID <= 0 {
		return nil, ErrInvalidOrderID
	}

	return s.syncOrdersWithResolver(ctx, trigger, func(ctx context.Context, summary *SyncSummary) ([]port.OrderSyncCommand, error) {
		return s.resolveCommandsByOrderID(ctx, summary, orderID)
	})
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
