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
	// ErrNilSource is returned when a nil Shopify customer source is provided.
	ErrNilSource = errors.New("shopify contact source must not be nil")
	// ErrNilTarget is returned when a nil contact target is provided.
	ErrNilTarget = errors.New("shopify contact target must not be nil")
	// ErrSyncDisabled is returned when contact sync is disabled.
	ErrSyncDisabled = errors.New("shopify contact sync is disabled")
	// ErrInvalidCustomerID is returned when Shopify customer identifiers are empty.
	ErrInvalidCustomerID = errors.New("shopify customer id is required")
	// ErrContactNotFound is returned when Shopify customers are not found.
	ErrContactNotFound = errors.New("shopify contact not found")
	// ErrIntegrationUnavailable is returned when Shopify is unavailable.
	ErrIntegrationUnavailable = errors.New("shopify integration is unavailable")
)

// CircuitBreaker defines optional dependency circuit-breaker behavior.
type CircuitBreaker interface {
	// Execute runs one function through the breaker.
	Execute(fn func() error) error
}

// CircuitBreakers defines optional breaker wiring for contact synchronization.
type CircuitBreakers struct {
	// Source defines breaker behavior for Shopify source requests.
	Source CircuitBreaker
}

// SyncConfig defines targeted contact synchronization configuration values.
type SyncConfig struct {
	// Enabled reports whether Shopify contact sync is enabled.
	Enabled bool
}

// SyncSummary defines targeted contact sync output values.
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
	// ContactID defines resolved mainstream contact identifiers.
	ContactID string `json:"contactId,omitempty"`
}

// Service defines Shopify contact synchronization behavior.
type Service interface {
	// ValidateIntegration verifies source connectivity and credentials.
	ValidateIntegration(ctx context.Context) error
	// SyncContacts synchronizes all Shopify customers for the active installation.
	SyncContacts(ctx context.Context, trigger string) (*SyncSummary, error)
	// SyncContactByID synchronizes one Shopify customer by identifier.
	SyncContactByID(ctx context.Context, trigger string, id string) (*SyncSummary, error)
	// SetSyncRecorder configures sync-run recording behavior.
	SetSyncRecorder(recorder shopifyport.SyncRecorder)
}

// ContactSyncService defines Shopify contact synchronization behavior.
type ContactSyncService struct {
	// cfg defines feature configuration values.
	cfg SyncConfig
	// source defines Shopify customer source dependencies.
	source shopifyport.CustomerSource
	// target defines mainstream contact upsert dependencies.
	target shopifyport.ContactSyncTarget
	// logger defines structured logging dependencies.
	logger *zap.Logger
	// recorder defines sync-run recording behavior.
	recorder shopifyport.SyncRecorder
	// sourceBreaker defines optional Shopify source breaker behavior.
	sourceBreaker CircuitBreaker
}

var (
	// _ ensures ContactSyncService satisfies Service contracts.
	_ Service = (*ContactSyncService)(nil)
)

// NewService creates Shopify contact synchronization services.
func NewService(cfg SyncConfig, source shopifyport.CustomerSource, target shopifyport.ContactSyncTarget, providedLogger *zap.Logger, breakers ...CircuitBreakers) (*ContactSyncService, error) {
	if source == nil {
		return nil, ErrNilSource
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

	return &ContactSyncService{
		cfg:           cfg,
		source:        source,
		target:        target,
		logger:        logger,
		recorder:      shopifyport.NoopSyncRecorder{},
		sourceBreaker: resolvedBreaker.Source,
	}, nil
}

// SetSyncRecorder configures sync-run recording behavior.
func (s *ContactSyncService) SetSyncRecorder(recorder shopifyport.SyncRecorder) {
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
func (s *ContactSyncService) ValidateIntegration(ctx context.Context) error {
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

// SyncContactByID synchronizes one Shopify customer by identifier.
func (s *ContactSyncService) SyncContactByID(ctx context.Context, trigger string, id string) (*SyncSummary, error) {
	if !s.cfg.Enabled {
		return nil, ErrSyncDisabled
	}
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, ErrInvalidCustomerID
	}

	resolvedTrigger := resolveTrigger(trigger)
	runID, _ := s.recorder.StartRun(ctx, "shopify.contacts", resolvedTrigger)
	customer, err := s.loadCustomer(ctx, trimmedID)
	if err != nil {
		s.recordFailure(ctx, runID, trimmedID, err)
		return nil, err
	}

	entity, err := s.target.UpsertContact(ctx, BuildContactSyncCommand(customer))
	if err != nil {
		s.recordFailure(ctx, runID, trimmedID, err)
		return nil, err
	}

	summary := &SyncSummary{
		RunID:     runID,
		Trigger:   resolvedTrigger,
		Processed: 1,
		Succeeded: 1,
		ContactID: strings.TrimSpace(entity.ID),
	}
	if completeErr := s.recorder.CompleteRun(ctx, runID, summary.Processed, summary.Succeeded, summary.Failed, summary.Skipped); completeErr != nil {
		s.logger.Warn("complete shopify contact sync run failed", zap.Error(completeErr))
	}

	return summary, nil
}

// SyncContacts synchronizes all Shopify customers for the active installation.
func (s *ContactSyncService) SyncContacts(ctx context.Context, trigger string) (*SyncSummary, error) {
	if !s.cfg.Enabled {
		return nil, ErrSyncDisabled
	}

	const pageSize = 250
	resolvedTrigger := resolveTrigger(trigger)
	runID, _ := s.recorder.StartRun(ctx, "shopify.contacts", resolvedTrigger)
	summary := &SyncSummary{RunID: runID, Trigger: resolvedTrigger}

	sinceID := ""
	for {
		if err := ctx.Err(); err != nil {
			_ = s.recorder.FailRun(ctx, runID, summary.Processed, summary.Succeeded, summary.Failed, summary.Skipped, nil)
			return nil, err
		}

		var customers []shopifyport.ShopifyCustomer
		var hasMore bool
		err := s.executeWithBreaker(s.sourceBreaker, ErrIntegrationUnavailable, func() error {
			var listErr error
			customers, hasMore, listErr = s.source.ListCustomers(ctx, sinceID, pageSize)
			return listErr
		})
		if err != nil {
			_ = s.recorder.FailRun(ctx, runID, summary.Processed, summary.Succeeded, summary.Failed, summary.Skipped, nil)
			return nil, fmt.Errorf("%w: %v", ErrIntegrationUnavailable, err)
		}

		for _, customer := range customers {
			summary.Processed++
			if _, upsertErr := s.target.UpsertContact(ctx, BuildContactSyncCommand(customer)); upsertErr != nil {
				summary.Failed++
				s.logger.Warn("shopify contact sync failed", zap.String("id", customer.ID), zap.Error(upsertErr))
			} else {
				summary.Succeeded++
			}
		}

		if len(customers) > 0 {
			sinceID = customers[len(customers)-1].ID
		}

		if !hasMore {
			break
		}
	}

	if completeErr := s.recorder.CompleteRun(ctx, runID, summary.Processed, summary.Succeeded, summary.Failed, summary.Skipped); completeErr != nil {
		s.logger.Warn("complete shopify contacts sync run failed", zap.Error(completeErr))
	}

	return summary, nil
}

func (s *ContactSyncService) loadCustomer(ctx context.Context, id string) (shopifyport.ShopifyCustomer, error) {
	var customer shopifyport.ShopifyCustomer
	err := s.executeWithBreaker(s.sourceBreaker, ErrIntegrationUnavailable, func() error {
		var sourceErr error
		customer, sourceErr = s.source.GetCustomer(ctx, id)
		return sourceErr
	})
	if err == nil {
		return customer, nil
	}
	if errors.Is(err, shopifyport.ErrCustomerNotFound) {
		return shopifyport.ShopifyCustomer{}, ErrContactNotFound
	}
	if errors.Is(err, ErrIntegrationUnavailable) {
		return shopifyport.ShopifyCustomer{}, err
	}

	return shopifyport.ShopifyCustomer{}, fmt.Errorf("%w: %v", ErrIntegrationUnavailable, err)
}

func (s *ContactSyncService) recordFailure(ctx context.Context, runID string, id string, err error) {
	syncErr := shopifyport.SyncError{Type: "contact", Code: strings.TrimSpace(id), Message: err.Error()}
	if failErr := s.recorder.FailRun(ctx, runID, 1, 0, 1, 0, []shopifyport.SyncError{syncErr}); failErr != nil {
		s.logger.Warn("fail shopify contact sync run failed", zap.Error(failErr))
	}
}

func (s *ContactSyncService) executeWithBreaker(breaker CircuitBreaker, unavailableErr error, fn func() error) error {
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

func resolveTrigger(trigger string) string {
	trimmed := strings.TrimSpace(trigger)
	if trimmed != "" {
		return trimmed
	}

	return "manual"
}
