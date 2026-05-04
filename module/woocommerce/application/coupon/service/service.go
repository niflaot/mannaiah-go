package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	woocouponevent "mannaiah/module/woocommerce/application/coupon/event"
	"mannaiah/module/woocommerce/port"

	"go.uber.org/zap"
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
	// ErrPartialSyncFailure is returned when coupon sync finishes with one or more failed upserts.
	ErrPartialSyncFailure = errors.New("woocommerce coupon sync completed with failed items")
)

const maxRecordedCouponSyncErrors = 25

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
	// ValidateIntegration verifies sync preconditions and WooCommerce connectivity.
	ValidateIntegration(ctx context.Context) error
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
	// syncRecorder defines optional sync-run recording dependencies.
	syncRecorder port.SyncRecorder
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
		syncRecorder:  port.NoopSyncRecorder{},
	}, nil
}

// SetSyncRecorder configures optional sync run recording dependencies.
func (s *CouponSyncService) SetSyncRecorder(recorder port.SyncRecorder) {
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
func (s *CouponSyncService) ValidateIntegration(ctx context.Context) error {
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

// SyncCoupons performs coupon synchronization and emits integration events.
func (s *CouponSyncService) SyncCoupons(ctx context.Context, trigger string) (*SyncSummary, error) {
	summary := &SyncSummary{Trigger: trigger}
	runID := s.startSyncRunRecord(ctx, summary.Trigger)
	syncErrors := make([]port.SyncError, 0, maxRecordedCouponSyncErrors)

	_ = s.publisher.Publish(ctx, woocouponevent.NewSyncStartedEvent(trigger))

	if err := s.ValidateIntegration(ctx); err != nil {
		_ = s.publisher.Publish(ctx, woocouponevent.NewSyncFailedEvent(toEventSummary(summary), err))
		s.finishSyncRunRecord(ctx, runID, summary, err, nil)
		return nil, err
	}

	if err := s.syncAllPages(ctx, summary, &syncErrors); err != nil {
		_ = s.publisher.Publish(ctx, woocouponevent.NewSyncFailedEvent(toEventSummary(summary), err))
		s.finishSyncRunRecord(ctx, runID, summary, err, syncErrors)
		return summary, fmt.Errorf("woocommerce coupon sync: %w", err)
	}

	if err := buildPartialSyncFailure(summary); err != nil {
		_ = s.publisher.Publish(ctx, woocouponevent.NewSyncFailedEvent(toEventSummary(summary), err))
		s.finishSyncRunRecord(ctx, runID, summary, err, syncErrors)
		return summary, fmt.Errorf("woocommerce coupon sync: %w", err)
	}

	_ = s.publisher.Publish(ctx, woocouponevent.NewSyncCompletedEvent(toEventSummary(summary)))
	s.finishSyncRunRecord(ctx, runID, summary, nil, nil)
	return summary, nil
}

// syncAllPages pages through WooCommerce coupons and upserts each one.
func (s *CouponSyncService) syncAllPages(ctx context.Context, summary *SyncSummary, syncErrors *[]port.SyncError) error {
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
				appendCouponSyncError(syncErrors, coupon, upsertErr)
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

// buildPartialSyncFailure returns a stable sync error when row failures occurred.
func buildPartialSyncFailure(summary *SyncSummary) error {
	if summary == nil || summary.Failed == 0 {
		return nil
	}

	succeeded := summary.Created + summary.Updated + summary.Unchanged

	return fmt.Errorf(
		"%w: processed=%d failed=%d succeeded=%d skipped=%d",
		ErrPartialSyncFailure,
		summary.Processed,
		summary.Failed,
		succeeded,
		summary.Skipped,
	)
}

// appendCouponSyncError stores a bounded sync error entry for failed coupon upserts.
func appendCouponSyncError(syncErrors *[]port.SyncError, coupon port.WooCoupon, err error) {
	if syncErrors == nil || err == nil || len(*syncErrors) >= maxRecordedCouponSyncErrors {
		return
	}

	message := fmt.Sprintf("coupon %d: %v", coupon.ID, err)
	if code := strings.TrimSpace(coupon.Code); code != "" {
		message = fmt.Sprintf("coupon %d (%s): %v", coupon.ID, code, err)
	}

	*syncErrors = append(*syncErrors, port.SyncError{
		Type:    "upsert",
		Code:    "coupon_upsert_failed",
		Message: message,
	})
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
