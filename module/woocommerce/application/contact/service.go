package contact

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"go.uber.org/zap"
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
}

// upsertResult defines command upsert result payload values.
type upsertResult struct {
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
func NewService(cfg SyncConfig, source port.OrderSource, target port.ContactSyncTarget, publisher port.IntegrationEventPublisher, providedLogger *zap.Logger) (*ContactSyncService, error) {
	if source == nil {
		return nil, ErrNilSource
	}
	if target == nil {
		return nil, ErrNilTarget
	}

	return &ContactSyncService{
		source:    source,
		target:    target,
		publisher: resolvePublisher(publisher),
		logger:    resolveLogger(providedLogger),
		cfg:       normalizeSyncConfig(cfg),
	}, nil
}

// ValidateIntegration verifies sync preconditions and WooCommerce connectivity.
func (s *ContactSyncService) ValidateIntegration(ctx context.Context) error {
	if !s.cfg.Enabled {
		return ErrSyncDisabled
	}

	if err := s.source.Validate(ctx); err != nil {
		return fmt.Errorf("%w: %v", ErrIntegrationUnavailable, err)
	}

	return nil
}

// SyncContacts performs contact synchronization and emits integration events.
func (s *ContactSyncService) SyncContacts(ctx context.Context, trigger string) (*SyncSummary, error) {
	summary := &SyncSummary{Trigger: normalizeTrigger(trigger)}
	s.publishEvent(ctx, buildSyncStartedEvent(summary.Trigger))

	if err := s.ValidateIntegration(ctx); err != nil {
		s.publishEvent(ctx, buildSyncFailedEvent(*summary, err))
		return nil, err
	}

	seenEmails := map[string]struct{}{}
	page := 1
	for {
		if err := ctx.Err(); err != nil {
			s.publishEvent(ctx, buildSyncFailedEvent(*summary, err))
			return nil, err
		}

		orders, hasNext, err := s.source.ListOrders(ctx, page, s.cfg.PageSize)
		if err != nil {
			wrappedErr := fmt.Errorf("list woocommerce orders page %d: %w", page, err)
			s.publishEvent(ctx, buildSyncFailedEvent(*summary, wrappedErr))
			return nil, wrappedErr
		}

		if len(orders) == 0 {
			break
		}

		if err := s.processPage(ctx, orders, seenEmails, summary); err != nil {
			s.publishEvent(ctx, buildSyncFailedEvent(*summary, err))
			return nil, err
		}

		if !hasNext {
			break
		}
		page++
	}

	s.publishEvent(ctx, buildSyncCompletedEvent(*summary))
	return summary, nil
}

// resolveLogger resolves nil loggers to no-op defaults.
func resolveLogger(providedLogger *zap.Logger) *zap.Logger {
	if providedLogger != nil {
		return providedLogger
	}

	return zap.NewNop()
}

// normalizeSyncConfig normalizes sync config defaults.
func normalizeSyncConfig(cfg SyncConfig) SyncConfig {
	resolved := cfg
	if resolved.PageSize <= 0 {
		resolved.PageSize = 100
	}
	if resolved.WorkerCount <= 0 {
		resolved.WorkerCount = 8
	}

	return resolved
}

// normalizeTrigger resolves sync trigger fallback values.
func normalizeTrigger(trigger string) string {
	resolved := strings.TrimSpace(trigger)
	if resolved == "" {
		return "manual"
	}

	return resolved
}

// processPage applies concurrent upsert behavior for a WooCommerce order page.
func (s *ContactSyncService) processPage(ctx context.Context, orders []port.WooOrder, seenEmails map[string]struct{}, summary *SyncSummary) error {
	commands := make([]port.ContactSyncCommand, 0, len(orders))

	for _, order := range orders {
		command, shouldProcess := mapOrderToCommand(order)
		if !shouldProcess {
			summary.Skipped++
			continue
		}

		emailKey := strings.ToLower(strings.TrimSpace(command.Email))
		if _, seen := seenEmails[emailKey]; seen {
			summary.Skipped++
			continue
		}
		seenEmails[emailKey] = struct{}{}
		commands = append(commands, command)
	}

	if len(commands) == 0 {
		return nil
	}

	workerCount := s.cfg.WorkerCount
	if workerCount > len(commands) {
		workerCount = len(commands)
	}

	workChannel := make(chan port.ContactSyncCommand, len(commands))
	resultChannel := make(chan upsertResult, len(commands))
	var workerWait sync.WaitGroup

	for workerIndex := 0; workerIndex < workerCount; workerIndex++ {
		workerWait.Add(1)
		go func() {
			defer workerWait.Done()
			for command := range workChannel {
				if err := ctx.Err(); err != nil {
					resultChannel <- upsertResult{err: err}
					continue
				}

				outcome, upsertErr := s.target.UpsertByEmail(ctx, command)
				resultChannel <- upsertResult{outcome: outcome, err: upsertErr}
			}
		}()
	}

	for _, command := range commands {
		if err := ctx.Err(); err != nil {
			close(workChannel)
			workerWait.Wait()
			close(resultChannel)
			return err
		}
		workChannel <- command
	}
	close(workChannel)

	workerWait.Wait()
	close(resultChannel)

	for result := range resultChannel {
		if errors.Is(result.err, context.Canceled) || errors.Is(result.err, context.DeadlineExceeded) {
			return result.err
		}

		summary.Processed++
		if result.err != nil {
			summary.Failed++
			s.logger.Warn("woocommerce contact sync upsert failed", zap.Error(result.err))
			continue
		}

		applyOutcome(summary, result.outcome)
	}

	return nil
}

// applyOutcome applies upsert outcomes to sync summary counters.
func applyOutcome(summary *SyncSummary, outcome port.UpsertOutcome) {
	switch outcome {
	case port.UpsertOutcomeCreated:
		summary.Created++
	case port.UpsertOutcomeUnchanged:
		summary.Unchanged++
	default:
		summary.Updated++
	}
}

// mapOrderToCommand maps WooCommerce orders into contact upsert command values.
func mapOrderToCommand(order port.WooOrder) (port.ContactSyncCommand, bool) {
	email := strings.ToLower(strings.TrimSpace(order.BillingEmail))
	if email == "" {
		return port.ContactSyncCommand{}, false
	}

	firstName := strings.TrimSpace(order.BillingFirstName)
	lastName := strings.TrimSpace(order.BillingLastName)
	if firstName == "" || lastName == "" {
		return port.ContactSyncCommand{}, false
	}

	documentNumber := mapDocumentNumber(order.Metadata)
	documentType := ""
	if documentNumber != "" {
		documentType = "CC"
	}

	return port.ContactSyncCommand{
		Email:          email,
		FirstName:      firstName,
		LastName:       lastName,
		Phone:          normalizePhone(order.BillingPhone),
		Address:        strings.TrimSpace(order.BillingAddress1),
		AddressExtra:   strings.TrimSpace(order.BillingAddress2),
		CityCode:       strings.TrimSpace(order.BillingCity),
		DocumentType:   documentType,
		DocumentNumber: documentNumber,
	}, true
}

// mapDocumentNumber resolves WooCommerce billing document metadata values.
func mapDocumentNumber(metadata map[string]string) string {
	if len(metadata) == 0 {
		return ""
	}

	return strings.TrimSpace(metadata[billingDocumentMetaKey])
}

// normalizePhone normalizes WooCommerce phone values to +57-prefixed values.
func normalizePhone(value string) string {
	normalized := strings.ReplaceAll(strings.TrimSpace(value), " ", "")
	normalized = strings.ReplaceAll(normalized, "+", "")
	normalized = strings.TrimPrefix(normalized, "57")
	if normalized == "" {
		return ""
	}

	return "+57" + normalized
}
