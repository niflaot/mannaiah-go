package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"
	shopifyport "mannaiah/module/shopify/port"
)

var (
	// ErrNilSource is returned when a nil Shopify order source is provided.
	ErrNilSource = errors.New("shopify order source must not be nil")
	// ErrNilContactTarget is returned when a nil contact sync target is provided.
	ErrNilContactTarget = errors.New("shopify order contact target must not be nil")
	// ErrNilTarget is returned when a nil order sync target is provided.
	ErrNilTarget = errors.New("shopify order target must not be nil")
	// ErrSyncDisabled is returned when order sync is disabled.
	ErrSyncDisabled = errors.New("shopify order sync is disabled")
	// ErrInvalidOrderID is returned when Shopify order identifiers are empty.
	ErrInvalidOrderID = errors.New("shopify order id is required")
	// ErrOrderNotFound is returned when Shopify orders are not found.
	ErrOrderNotFound = errors.New("shopify order not found")
	// ErrIntegrationUnavailable is returned when Shopify is unavailable.
	ErrIntegrationUnavailable = errors.New("shopify integration is unavailable")
	// ErrOrderContactEmailRequired is returned when order payloads do not contain a contact email.
	ErrOrderContactEmailRequired = errors.New("shopify order contact email is required")
)

// CircuitBreaker defines optional dependency circuit-breaker behavior.
type CircuitBreaker interface {
	// Execute runs one function through the breaker.
	Execute(fn func() error) error
}

// CircuitBreakers defines optional breaker wiring for order synchronization.
type CircuitBreakers struct {
	// Source defines breaker behavior for Shopify source requests.
	Source CircuitBreaker
	// Destination defines breaker behavior for outbound Shopify writes.
	Destination CircuitBreaker
}

// SyncConfig defines targeted order synchronization configuration values.
type SyncConfig struct {
	// Enabled reports whether Shopify order sync is enabled.
	Enabled bool
	// Realm defines the mainstream order realm value used for Shopify orders.
	Realm string
}

// SyncSummary defines targeted order sync output values.
type SyncSummary struct {
	// RunID defines optional sync-run identifiers.
	RunID string `json:"runId,omitempty"`
	// Trigger defines sync trigger values.
	Trigger string `json:"trigger"`
	// Processed defines processed record counts.
	Processed int `json:"processed"`
	// Succeeded defines success counts.
	Succeeded int `json:"succeeded"`
	// Failed defines failed record counts.
	Failed int `json:"failed"`
	// Skipped defines skipped record counts.
	Skipped int `json:"skipped"`
	// OrderID defines resolved mainstream order identifiers.
	OrderID string `json:"orderId,omitempty"`
	// ContactID defines resolved mainstream contact identifiers.
	ContactID string `json:"contactId,omitempty"`
}

// Service defines Shopify order synchronization behavior.
type Service interface {
	// ValidateIntegration verifies source connectivity and credentials.
	ValidateIntegration(ctx context.Context) error
	// SyncOrders synchronizes all Shopify orders for the active installation.
	SyncOrders(ctx context.Context, trigger string) (*SyncSummary, error)
	// SyncOrderByID synchronizes one Shopify order by identifier.
	SyncOrderByID(ctx context.Context, trigger string, id string) (*SyncSummary, error)
	// SetSyncRecorder configures sync-run recording behavior.
	SetSyncRecorder(recorder shopifyport.SyncRecorder)
}

// OrderSyncService defines Shopify order synchronization behavior.
type OrderSyncService struct {
	// cfg defines feature configuration values.
	cfg SyncConfig
	// source defines Shopify order source dependencies.
	source shopifyport.OrderSource
	// contactTarget defines contact upsert behavior used prior to order upsert.
	contactTarget shopifyport.ContactSyncTarget
	// target defines mainstream order upsert behavior.
	target shopifyport.OrderSyncTarget
	// logger defines structured logging dependencies.
	logger *zap.Logger
	// recorder defines sync-run recording behavior.
	recorder shopifyport.SyncRecorder
	// sourceBreaker defines optional Shopify source breaker behavior.
	sourceBreaker CircuitBreaker
}

var (
	// _ ensures OrderSyncService satisfies Service contracts.
	_ Service = (*OrderSyncService)(nil)
)

// NewService creates Shopify order synchronization services.
func NewService(cfg SyncConfig, source shopifyport.OrderSource, contactTarget shopifyport.ContactSyncTarget, target shopifyport.OrderSyncTarget, providedLogger *zap.Logger, breakers ...CircuitBreakers) (*OrderSyncService, error) {
	if source == nil {
		return nil, ErrNilSource
	}
	if contactTarget == nil {
		return nil, ErrNilContactTarget
	}
	if target == nil {
		return nil, ErrNilTarget
	}

	resolvedBreaker := CircuitBreakers{}
	if len(breakers) > 0 {
		resolvedBreaker = breakers[0]
	}
	logger := providedLogger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &OrderSyncService{
		cfg:           cfg,
		source:        source,
		contactTarget: contactTarget,
		target:        target,
		logger:        logger,
		recorder:      shopifyport.NoopSyncRecorder{},
		sourceBreaker: resolvedBreaker.Source,
	}, nil
}

// SetSyncRecorder configures sync-run recording behavior.
func (s *OrderSyncService) SetSyncRecorder(recorder shopifyport.SyncRecorder) {
	if s == nil {
		return
	}
	if recorder == nil {
		s.recorder = shopifyport.NoopSyncRecorder{}
		return
	}

	s.recorder = recorder
}

// ValidateIntegration verifies source connectivity and credentials.
func (s *OrderSyncService) ValidateIntegration(ctx context.Context) error {
	if s == nil {
		return ErrIntegrationUnavailable
	}
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

// SyncOrderByID synchronizes one Shopify order by identifier.
func (s *OrderSyncService) SyncOrderByID(ctx context.Context, trigger string, id string) (*SyncSummary, error) {
	if !s.cfg.Enabled {
		return nil, ErrSyncDisabled
	}
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, ErrInvalidOrderID
	}

	resolvedTrigger := resolveTrigger(trigger)
	runID, err := s.recorder.StartRun(ctx, "shopify.orders", resolvedTrigger)
	if err != nil {
		s.logger.Warn("start shopify order sync run failed", zap.Error(err))
	}
	order, err := s.loadOrder(ctx, trimmedID)
	if err != nil {
		s.recordFailure(ctx, runID, trimmedID, err)
		return nil, err
	}

	contact, err := s.syncOrderContact(ctx, order)
	if err != nil {
		s.recordFailure(ctx, runID, trimmedID, err)
		return nil, err
	}

	entity, err := s.target.UpsertOrder(ctx, BuildOrderSyncCommand(order, contact.ID, s.cfg.Realm, resolvedTrigger))
	if err != nil {
		s.recordFailure(ctx, runID, trimmedID, err)
		return nil, err
	}

	summary := &SyncSummary{
		RunID:     runID,
		Trigger:   resolvedTrigger,
		Processed: 1,
		Succeeded: 1,
		OrderID:   strings.TrimSpace(entity.ID),
		ContactID: strings.TrimSpace(contact.ID),
	}
	if completeErr := s.recorder.CompleteRun(ctx, runID, summary.Processed, summary.Succeeded, summary.Failed, summary.Skipped); completeErr != nil {
		s.logger.Warn("complete shopify order sync run failed", zap.Error(completeErr))
	}

	return summary, nil
}

// SyncOrders synchronizes all Shopify orders for the active installation.
func (s *OrderSyncService) SyncOrders(ctx context.Context, trigger string) (*SyncSummary, error) {
	if !s.cfg.Enabled {
		return nil, ErrSyncDisabled
	}

	const pageSize = 250
	resolvedTrigger := resolveTrigger(trigger)
	runID, startErr := s.recorder.StartRun(ctx, "shopify.orders", resolvedTrigger)
	if startErr != nil {
		s.logger.Warn("start shopify orders sync run failed", zap.Error(startErr))
	}
	summary := &SyncSummary{RunID: runID, Trigger: resolvedTrigger}

	sinceID := ""
	for {
		if err := ctx.Err(); err != nil {
			_ = s.recorder.FailRun(ctx, runID, summary.Processed, summary.Succeeded, summary.Failed, summary.Skipped, nil)
			return nil, err
		}

		var orders []shopifyport.ShopifyOrder
		var hasMore bool
		err := s.executeWithBreaker(s.sourceBreaker, ErrIntegrationUnavailable, func() error {
			var listErr error
			orders, hasMore, listErr = s.source.ListOrders(ctx, sinceID, pageSize)
			return listErr
		})
		if err != nil {
			_ = s.recorder.FailRun(ctx, runID, summary.Processed, summary.Succeeded, summary.Failed, summary.Skipped, nil)
			return nil, fmt.Errorf("%w: %v", ErrIntegrationUnavailable, err)
		}

		for _, order := range orders {
			summary.Processed++
			contact, contactErr := s.syncOrderContact(ctx, order)
			if contactErr != nil {
				summary.Failed++
				s.logger.Warn("shopify order contact sync failed", zap.String("id", order.ID), zap.Error(contactErr))
				continue
			}
			if _, upsertErr := s.target.UpsertOrder(ctx, BuildOrderSyncCommand(order, contact.ID, s.cfg.Realm, resolvedTrigger)); upsertErr != nil {
				summary.Failed++
				s.logger.Warn("shopify order sync failed", zap.String("id", order.ID), zap.Error(upsertErr))
			} else {
				summary.Succeeded++
			}
		}

		if len(orders) > 0 {
			sinceID = orders[len(orders)-1].ID
		}

		if !hasMore {
			break
		}
	}

	if completeErr := s.recorder.CompleteRun(ctx, runID, summary.Processed, summary.Succeeded, summary.Failed, summary.Skipped); completeErr != nil {
		s.logger.Warn("complete shopify orders sync run failed", zap.Error(completeErr))
	}

	return summary, nil
}

func (s *OrderSyncService) loadOrder(ctx context.Context, id string) (shopifyport.ShopifyOrder, error) {
	var order shopifyport.ShopifyOrder
	err := s.executeWithBreaker(s.sourceBreaker, ErrIntegrationUnavailable, func() error {
		var sourceErr error
		order, sourceErr = s.source.GetOrder(ctx, id)
		return sourceErr
	})
	if err == nil {
		return order, nil
	}
	if errors.Is(err, shopifyport.ErrOrderNotFound) {
		return shopifyport.ShopifyOrder{}, ErrOrderNotFound
	}
	if errors.Is(err, ErrIntegrationUnavailable) {
		return shopifyport.ShopifyOrder{}, err
	}

	return shopifyport.ShopifyOrder{}, fmt.Errorf("%w: %v", ErrIntegrationUnavailable, err)
}

func (s *OrderSyncService) syncOrderContact(ctx context.Context, order shopifyport.ShopifyOrder) (*shopifyportContact, error) {
	command, err := BuildOrderContactSyncCommand(order)
	if err != nil {
		return nil, err
	}
	entity, err := s.contactTarget.UpsertContact(ctx, command)
	if err != nil {
		return nil, err
	}
	return &shopifyportContact{ID: strings.TrimSpace(entity.ID)}, nil
}

func (s *OrderSyncService) recordFailure(ctx context.Context, runID string, id string, err error) {
	syncErr := shopifyport.SyncError{Type: "order", Code: strings.TrimSpace(id), Message: err.Error()}
	if failErr := s.recorder.FailRun(ctx, runID, 1, 0, 1, 0, []shopifyport.SyncError{syncErr}); failErr != nil {
		s.logger.Warn("fail shopify order sync run failed", zap.Error(failErr))
	}
}

func (s *OrderSyncService) executeWithBreaker(breaker CircuitBreaker, unavailableErr error, fn func() error) error {
	if breaker == nil {
		return fn()
	}

	var operationErr error
	err := breaker.Execute(func() error {
		operationErr = fn()
		return operationErr
	})
	if err == nil {
		return nil
	}
	if operationErr != nil {
		return operationErr
	}

	return unavailableErr
}

type shopifyportContact struct {
	ID string
}

func resolveTrigger(trigger string) string {
	trimmed := strings.TrimSpace(trigger)
	if trimmed == "" {
		return "shopify_sync"
	}
	if strings.HasPrefix(trimmed, "shopify_") {
		return trimmed
	}

	return "shopify_" + trimmed
}
