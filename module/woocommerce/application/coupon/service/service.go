package service

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"
	woocouponevent "mannaiah/module/woocommerce/application/coupon/event"
	"mannaiah/module/woocommerce/port"
)

var (
	// ErrNilSource is returned when a nil WooCommerce coupon source is used.
	ErrNilSource = errors.New("woocommerce coupon source must not be nil")
	// ErrNilTarget is returned when a nil coupon sync target is used.
	ErrNilTarget = errors.New("woocommerce coupon sync target must not be nil")
	// ErrSyncDisabled is returned when coupon sync is disabled by configuration.
	ErrSyncDisabled = errors.New("woocommerce coupons sync is disabled")
	// ErrIntegrationUnavailable is returned when WooCommerce integration is unavailable.
	ErrIntegrationUnavailable = errors.New("woocommerce integration is unavailable")
	// ErrUpsertUnavailable is returned when coupon-upsert dependencies are unavailable.
	ErrUpsertUnavailable = errors.New("woocommerce coupon upsert dependency is unavailable")
)

// SyncConfig defines coupon sync behavior configuration values.
type SyncConfig struct {
	// Enabled defines whether coupon sync behavior is enabled.
	Enabled bool
	// PageSize defines WooCommerce coupon page sizes.
	PageSize int
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
}

// SyncSummary defines coupon sync execution results.
type SyncSummary struct {
	// Trigger defines sync trigger names.
	Trigger string `json:"trigger"`
	// Processed defines upsert-attempt counts.
	Processed int `json:"processed"`
	// Created defines created coupon counts.
	Created int `json:"created"`
	// Updated defines updated coupon counts.
	Updated int `json:"updated"`
	// Unchanged defines no-op coupon counts.
	Unchanged int `json:"unchanged"`
	// Skipped defines skipped-coupon counts.
	Skipped int `json:"skipped"`
	// Failed defines failed upsert counts.
	Failed int `json:"failed"`
}

// Service defines WooCommerce coupon sync use-case behavior.
type Service interface {
	// SyncCoupons performs coupon synchronization and emits integration events.
	SyncCoupons(ctx context.Context, trigger string) (*SyncSummary, error)
}

// CouponSyncService defines WooCommerce coupon sync use-case dependencies.
type CouponSyncService struct {
	// source defines WooCommerce coupon retrieval dependencies.
	source port.CouponSource
	// target defines coupon upsert dependencies.
	target port.CouponSyncTarget
	// publisher defines integration event publication dependencies.
	publisher port.IntegrationEventPublisher
	// logger defines structured log dependencies.
	logger *zap.Logger
	// cfg defines sync behavior configuration values.
	cfg SyncConfig
	// sourceBreaker guards WooCommerce API calls.
	sourceBreaker CircuitBreaker
}

var (
	// _ ensures CouponSyncService satisfies service contracts.
	_ Service = (*CouponSyncService)(nil)
)

// NewService creates WooCommerce coupon sync services.
func NewService(cfg SyncConfig, source port.CouponSource, target port.CouponSyncTarget, publisher port.IntegrationEventPublisher, providedLogger *zap.Logger, breakers ...CircuitBreakers) (*CouponSyncService, error) {
	if source == nil {
		return nil, ErrNilSource
	}
	if target == nil {
		return nil, ErrNilTarget
	}

	resolvedBreakers := resolveCircuitBreakers(breakers)

	return &CouponSyncService{
		source:        source,
		target:        target,
		publisher:     woocouponevent.ResolvePublisher(publisher),
		logger:        resolveLogger(providedLogger),
		cfg:           normalizeSyncConfig(cfg),
		sourceBreaker: resolvedBreakers.Source,
	}, nil
}

// SyncCoupons performs coupon synchronization and emits integration events.
func (s *CouponSyncService) SyncCoupons(ctx context.Context, trigger string) (*SyncSummary, error) {
	if !s.cfg.Enabled {
		return nil, ErrSyncDisabled
	}

	summary := &SyncSummary{Trigger: trigger}

	_ = s.publisher.Publish(ctx, woocouponevent.NewSyncStartedEvent(trigger))

	if err := s.syncAllPages(ctx, summary); err != nil {
		_ = s.publisher.Publish(ctx, woocouponevent.NewSyncFailedEvent(toEventSummary(summary), err))
		return summary, fmt.Errorf("woocommerce coupon sync: %w", err)
	}

	_ = s.publisher.Publish(ctx, woocouponevent.NewSyncCompletedEvent(toEventSummary(summary)))
	return summary, nil
}

// syncAllPages pages through WooCommerce coupons and upserts each one.
func (s *CouponSyncService) syncAllPages(ctx context.Context, summary *SyncSummary) error {
	page := 1
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		var coupons []port.WooCoupon
		var hasNext bool

		err := s.executeWithBreaker(s.sourceBreaker, ErrIntegrationUnavailable, func() error {
			var listErr error
			coupons, hasNext, listErr = s.source.ListCoupons(ctx, page, s.cfg.PageSize)
			return listErr
		})
		if err != nil {
			return fmt.Errorf("list woocommerce coupons page %d: %w", page, err)
		}

		for _, coupon := range coupons {
			if err := ctx.Err(); err != nil {
				return err
			}

			outcome, upsertErr := s.target.UpsertByWooID(ctx, coupon)
			summary.Processed++
			if upsertErr != nil {
				summary.Failed++
				s.logger.Warn("woocommerce coupon sync upsert failed",
					zap.Int("woo_coupon_id", coupon.ID),
					zap.String("code", coupon.Code),
					zap.Error(upsertErr),
				)
				continue
			}

			switch outcome {
			case port.UpsertOutcomeCreated:
				summary.Created++
			case port.UpsertOutcomeUnchanged:
				summary.Unchanged++
			default:
				summary.Updated++
			}
		}

		if !hasNext {
			break
		}
		page++
	}

	return nil
}

// executeWithBreaker runs an operation with optional circuit-breaker protection.
func (s *CouponSyncService) executeWithBreaker(breaker CircuitBreaker, openErr error, operation func() error) error {
	if breaker == nil {
		return operation()
	}

	err := breaker.Execute(operation)
	if err == nil {
		return nil
	}
	if breaker.IsOpenError(err) {
		return openErr
	}

	return err
}

// toEventSummary maps sync summary values to event summary values.
func toEventSummary(summary *SyncSummary) woocouponevent.Summary {
	if summary == nil {
		return woocouponevent.Summary{}
	}

	return woocouponevent.Summary{
		Trigger:   summary.Trigger,
		Processed: summary.Processed,
		Created:   summary.Created,
		Updated:   summary.Updated,
		Unchanged: summary.Unchanged,
		Skipped:   summary.Skipped,
		Failed:    summary.Failed,
	}
}
