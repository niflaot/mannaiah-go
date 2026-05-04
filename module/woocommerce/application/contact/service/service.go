package service

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"
	woocontactevent "mannaiah/module/woocommerce/application/contact/event"
	"mannaiah/module/woocommerce/port"
)

const (
	// billingDocumentMetaKey defines WooCommerce order metadata keys that store billing document values.
	billingDocumentMetaKey = "_billing_document"
)

var (
	// ErrNilSource is returned when a nil WooCommerce order source is used.
	ErrNilSource = errors.New("woocommerce order source must not be nil")
	// ErrNilTarget is returned when a nil contact sync target is used.
	ErrNilTarget = errors.New("woocommerce contact sync target must not be nil")
	// ErrSyncDisabled is returned when contact sync is disabled by configuration.
	ErrSyncDisabled = errors.New("woocommerce contacts sync is disabled")
	// ErrIntegrationUnavailable is returned when WooCommerce integration is unavailable.
	ErrIntegrationUnavailable = errors.New("woocommerce integration is unavailable")
	// ErrUpsertUnavailable is returned when contact-upsert dependencies are unavailable.
	ErrUpsertUnavailable = errors.New("woocommerce contact upsert dependency is unavailable")
	// ErrInvalidEmail is returned when targeted contact-sync email values are invalid.
	ErrInvalidEmail = errors.New("woocommerce contact sync email is required")
	// ErrContactNotFound is returned when targeted contact-sync email values are not present in Woo orders.
	ErrContactNotFound = errors.New("woocommerce contact not found")
)

// SyncConfig defines sync behavior configuration values.
type SyncConfig struct {
	// Enabled defines whether contact sync behavior is enabled.
	Enabled bool
	// PageSize defines WooCommerce order page sizes.
	PageSize int
	// WorkerCount defines concurrent contact upsert workers.
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
	// Upsert guards contact-upsert calls.
	Upsert CircuitBreaker
}

// SyncSummary defines contact sync execution results.
type SyncSummary struct {
	// Trigger defines sync trigger names.
	Trigger string `json:"trigger"`
	// Processed defines upsert-attempt counts.
	Processed int `json:"processed"`
	// Created defines created contact counts.
	Created int `json:"created"`
	// Updated defines updated contact counts.
	Updated int `json:"updated"`
	// Unchanged defines no-op contact counts.
	Unchanged int `json:"unchanged"`
	// Skipped defines skipped-order counts.
	Skipped int `json:"skipped"`
	// Failed defines failed upsert counts.
	Failed int `json:"failed"`
}

// Service defines WooCommerce sync use-case behavior.
type Service interface {
	// ValidateIntegration verifies sync preconditions and WooCommerce connectivity.
	ValidateIntegration(ctx context.Context) error
	// SyncContacts performs contact synchronization and emits integration events.
	SyncContacts(ctx context.Context, trigger string) (*SyncSummary, error)
	// SyncContactByEmail performs targeted contact synchronization for one email and emits integration events.
	SyncContactByEmail(ctx context.Context, trigger string, email string) (*SyncSummary, error)
}

// ContactSyncService defines WooCommerce contact sync use-case dependencies.
type ContactSyncService struct {
	// source defines WooCommerce order retrieval dependencies.
	source port.OrderSource
	// target defines contact upsert dependencies.
	target port.ContactSyncTarget
	// publisher defines integration event publication dependencies.
	publisher port.IntegrationEventPublisher
	// logger defines structured log dependencies.
	logger *zap.Logger
	// cfg defines sync behavior configuration values.
	cfg SyncConfig
	// sourceBreaker guards WooCommerce API calls.
	sourceBreaker CircuitBreaker
	// upsertBreaker guards contact-upsert calls.
	upsertBreaker CircuitBreaker
	// syncRecorder defines optional sync-run recording dependencies.
	syncRecorder port.SyncRecorder
	// membershipStamper defines optional membership stamp dependencies.
	membershipStamper port.MembershipStamper
}

// upsertResult defines command upsert result payload values.
type upsertResult struct {
	// command defines source sync command values.
	command port.ContactSyncCommand
	// outcome defines upsert outcomes.
	outcome port.UpsertOutcome
	// err defines upsert execution errors.
	err error
}

var (
	// _ ensures ContactSyncService satisfies service contracts.
	_ Service = (*ContactSyncService)(nil)
)

// NewService creates WooCommerce contact sync services.
func NewService(cfg SyncConfig, source port.OrderSource, target port.ContactSyncTarget, publisher port.IntegrationEventPublisher, providedLogger *zap.Logger, breakers ...CircuitBreakers) (*ContactSyncService, error) {
	if source == nil {
		return nil, ErrNilSource
	}
	if target == nil {
		return nil, ErrNilTarget
	}

	resolvedBreakers := resolveCircuitBreakers(breakers)

	return &ContactSyncService{
		source:            source,
		target:            target,
		publisher:         woocontactevent.ResolvePublisher(publisher),
		logger:            resolveLogger(providedLogger),
		cfg:               normalizeSyncConfig(cfg),
		sourceBreaker:     resolvedBreakers.Source,
		upsertBreaker:     resolvedBreakers.Upsert,
		syncRecorder:      port.NoopSyncRecorder{},
		membershipStamper: port.NoopMembershipStamper{},
	}, nil
}

// SetSyncRecorder configures optional sync run recording dependencies.
func (s *ContactSyncService) SetSyncRecorder(recorder port.SyncRecorder) {
	if s == nil {
		return
	}
	if recorder == nil {
		s.syncRecorder = port.NoopSyncRecorder{}
		return
	}

	s.syncRecorder = recorder
}

// SetMembershipStamper configures optional membership stamp dependencies.
func (s *ContactSyncService) SetMembershipStamper(stamper port.MembershipStamper) {
	if s == nil {
		return
	}
	if stamper == nil {
		s.membershipStamper = port.NoopMembershipStamper{}
		return
	}

	s.membershipStamper = stamper
}

// ValidateIntegration verifies sync preconditions and WooCommerce connectivity.
func (s *ContactSyncService) ValidateIntegration(ctx context.Context) error {
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

// SyncContacts performs contact synchronization and emits integration events.
func (s *ContactSyncService) SyncContacts(ctx context.Context, trigger string) (*SyncSummary, error) {
	return s.syncContactsWithLoader(ctx, trigger, s.loadPage, false)
}

// SyncContactByEmail performs targeted contact synchronization for one email and emits integration events.
func (s *ContactSyncService) SyncContactByEmail(ctx context.Context, trigger string, email string) (*SyncSummary, error) {
	resolvedEmail := normalizeEmailKey(email)
	if resolvedEmail == "" {
		return nil, ErrInvalidEmail
	}

	return s.syncContactsWithLoader(ctx, trigger, s.newEmailPageLoader(resolvedEmail), true)
}

// syncContactsWithLoader performs contact synchronization using provided page-loader behavior.
func (s *ContactSyncService) syncContactsWithLoader(ctx context.Context, trigger string, loader pageLoader, requireCommands bool) (summary *SyncSummary, err error) {
	summary = &SyncSummary{Trigger: normalizeTrigger(trigger)}
	runID := s.startSyncRunRecord(ctx, summary.Trigger)
	defer func() {
		s.finishSyncRunRecord(ctx, runID, summary, err)
	}()

	s.publishEvent(ctx, woocontactevent.NewSyncStartedEvent(summary.Trigger))

	if err := s.ValidateIntegration(ctx); err != nil {
		s.publishEvent(ctx, woocontactevent.NewSyncFailedEvent(toEventSummary(*summary), err))
		return nil, err
	}

	commandIndexByEmail := map[string]int{}
	pendingCommands := make([]port.ContactSyncCommand, 0)
	page := 1
	for {
		if err := ctx.Err(); err != nil {
			s.publishEvent(ctx, woocontactevent.NewSyncFailedEvent(toEventSummary(*summary), err))
			return nil, err
		}

		orders, hasNext, err := loader(ctx, page)
		if err != nil {
			wrappedErr := fmt.Errorf("list woocommerce orders page %d (%s): %w", page, formatSyncProgress(summary), err)
			s.publishEvent(ctx, woocontactevent.NewSyncFailedEvent(toEventSummary(*summary), wrappedErr))
			return nil, wrappedErr
		}

		if len(orders) == 0 {
			break
		}

		pendingCommands = collectCommandsFromOrders(orders, commandIndexByEmail, pendingCommands, summary)

		if !hasNext {
			break
		}
		page++
	}
	if requireCommands && len(pendingCommands) == 0 {
		s.publishEvent(ctx, woocontactevent.NewSyncFailedEvent(toEventSummary(*summary), ErrContactNotFound))
		return nil, ErrContactNotFound
	}

	if err := s.processCommands(ctx, pendingCommands, summary); err != nil {
		wrappedErr := fmt.Errorf("process woocommerce orders sync (%s): %w", formatSyncProgress(summary), err)
		s.publishEvent(ctx, woocontactevent.NewSyncFailedEvent(toEventSummary(*summary), wrappedErr))
		return nil, wrappedErr
	}

	s.publishEvent(ctx, woocontactevent.NewSyncCompletedEvent(toEventSummary(*summary)))
	return summary, nil
}

// loadPage retrieves one WooCommerce order page with breaker protection.
func (s *ContactSyncService) loadPage(ctx context.Context, page int) (orders []port.WooOrder, hasNext bool, err error) {
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
