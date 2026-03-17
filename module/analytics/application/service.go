package application

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"mannaiah/module/analytics/domain"
	"mannaiah/module/analytics/port"
)

const (
	seedBatchSize = 1000
)

var (
	// ErrNilDB is returned when nil db dependencies are provided.
	ErrNilDB = errors.New("analytics db must not be nil")
	// ErrDisabled is returned when analytics are disabled by configuration.
	ErrDisabled = errors.New("analytics module is disabled")
	// ErrBackendUnavailable is returned when analytics backend dependencies are unavailable.
	ErrBackendUnavailable = errors.New("analytics backend is unavailable")
)

// Status defines analytics status response values.
type Status struct {
	// Enabled reports module enablement state.
	Enabled bool `json:"enabled"`
	// BackendHealthy reports backend healthcheck state.
	BackendHealthy bool `json:"backendHealthy"`
	// Error defines optional backend healthcheck errors.
	Error string `json:"error,omitempty"`
}

// SeedSummary defines analytics seed execution summary values.
type SeedSummary struct {
	// Contacts defines seeded contact row counts.
	Contacts int64 `json:"contacts"`
	// Orders defines seeded order row counts.
	Orders int64 `json:"orders"`
	// OrderItems defines seeded order-item row counts.
	OrderItems int64 `json:"orderItems"`
	// MembershipEvents defines seeded membership-event row counts.
	MembershipEvents int64 `json:"membershipEvents"`
	// CampaignEvents defines seeded campaign-event row counts.
	CampaignEvents int64 `json:"campaignEvents"`
}

// Service defines analytics use-case behavior.
type Service interface {
	// Status returns analytics runtime health values.
	Status(ctx context.Context) Status
	// Seed executes best-effort initial data seeding behavior.
	Seed(ctx context.Context) (*SeedSummary, error)
	// ResolveContacts resolves contact ids for analytical filters.
	ResolveContacts(ctx context.Context, filter domain.SegmentFilter, page int, limit int) ([]string, error)
	// CountContacts counts contact ids for analytical filters.
	CountContacts(ctx context.Context, filter domain.SegmentFilter) (int64, error)
	// IngestContacts ingests contact snapshot rows from integration events.
	IngestContacts(ctx context.Context, rows []port.ContactSnapshot) error
	// IngestOrders ingests order and order-item fact rows from integration events.
	IngestOrders(ctx context.Context, orders []port.OrderFact, items []port.OrderItemFact) error
	// IngestMembershipEvents ingests membership events from integration events.
	IngestMembershipEvents(ctx context.Context, rows []port.MembershipEvent) error
	// IngestCampaignEvents ingests campaign delivery events from integration events.
	IngestCampaignEvents(ctx context.Context, rows []port.CampaignEvent) error
}

// AnalyticsService implements analytics use-cases.
type AnalyticsService struct {
	// enabled defines analytics enablement state.
	enabled bool
	// db defines transactional database dependencies used for initial seed reads.
	db *gorm.DB
	// store defines analytics backend dependencies.
	store port.Store
	// taxonomyStore defines optional taxonomy upsert dependencies used during seed.
	taxonomyStore port.TaxonomyStore
	// syncRecorder defines optional sync run recording dependencies.
	syncRecorder port.SyncRecorder
}

var (
	// _ ensures AnalyticsService satisfies analytics service contracts.
	_ Service = (*AnalyticsService)(nil)
	// _ ensures AnalyticsService satisfies resolver contracts.
	_ port.Resolver = (*AnalyticsService)(nil)
)

// NewService creates analytics services.
func NewService(enabled bool, db *gorm.DB, store port.Store) (*AnalyticsService, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	return &AnalyticsService{enabled: enabled, db: db, store: store, syncRecorder: port.NoopSyncRecorder{}}, nil
}

// SetTaxonomyStore configures optional taxonomy store dependencies used during seed.
func (s *AnalyticsService) SetTaxonomyStore(taxonomyStore port.TaxonomyStore) {
	if s == nil {
		return
	}

	s.taxonomyStore = taxonomyStore
}

// SetSyncRecorder configures optional sync run recording dependencies.
func (s *AnalyticsService) SetSyncRecorder(recorder port.SyncRecorder) {
	if s == nil {
		return
	}
	if recorder == nil {
		s.syncRecorder = port.NoopSyncRecorder{}
		return
	}

	s.syncRecorder = recorder
}

// Status returns analytics runtime health values.
func (s *AnalyticsService) Status(ctx context.Context) Status {
	if !s.enabled {
		return Status{Enabled: false, BackendHealthy: false}
	}
	if s.store == nil {
		return Status{Enabled: true, BackendHealthy: false, Error: ErrBackendUnavailable.Error()}
	}

	if err := s.store.Ping(ctx); err != nil {
		return Status{Enabled: true, BackendHealthy: false, Error: err.Error()}
	}

	return Status{Enabled: true, BackendHealthy: true}
}

// ResolveContacts resolves contact ids for analytical filters.
func (s *AnalyticsService) ResolveContacts(ctx context.Context, filter domain.SegmentFilter, page int, limit int) ([]string, error) {
	if !s.enabled {
		return nil, ErrDisabled
	}
	if s.store == nil {
		return nil, ErrBackendUnavailable
	}

	rows, err := s.store.ResolveContacts(ctx, filter, page, limit)
	if err != nil {
		return nil, fmt.Errorf("resolve analytics contacts: %w", err)
	}

	return rows, nil
}

// CountContacts counts contact ids for analytical filters.
func (s *AnalyticsService) CountContacts(ctx context.Context, filter domain.SegmentFilter) (int64, error) {
	if !s.enabled {
		return 0, ErrDisabled
	}
	if s.store == nil {
		return 0, ErrBackendUnavailable
	}

	count, err := s.store.CountContacts(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("count analytics contacts: %w", err)
	}

	return count, nil
}

// IngestContacts ingests contact snapshot rows from integration events.
func (s *AnalyticsService) IngestContacts(ctx context.Context, rows []port.ContactSnapshot) error {
	if !s.enabled {
		return ErrDisabled
	}
	if s.store == nil {
		return ErrBackendUnavailable
	}
	if err := s.store.UpsertContacts(ctx, rows); err != nil {
		return fmt.Errorf("ingest contacts snapshot rows: %w", err)
	}

	return nil
}

// IngestOrders ingests order and order-item fact rows from integration events.
func (s *AnalyticsService) IngestOrders(ctx context.Context, orders []port.OrderFact, items []port.OrderItemFact) error {
	if !s.enabled {
		return ErrDisabled
	}
	if s.store == nil {
		return ErrBackendUnavailable
	}
	if err := s.store.UpsertOrders(ctx, orders); err != nil {
		return fmt.Errorf("ingest orders fact rows: %w", err)
	}
	if err := s.store.UpsertOrderItems(ctx, items); err != nil {
		return fmt.Errorf("ingest order items fact rows: %w", err)
	}

	return nil
}

// IngestMembershipEvents ingests membership events from integration events.
func (s *AnalyticsService) IngestMembershipEvents(ctx context.Context, rows []port.MembershipEvent) error {
	if !s.enabled {
		return ErrDisabled
	}
	if s.store == nil {
		return ErrBackendUnavailable
	}
	if err := s.store.InsertMembershipEvents(ctx, rows); err != nil {
		return fmt.Errorf("ingest membership events: %w", err)
	}

	return nil
}

// IngestCampaignEvents ingests campaign delivery events from integration events.
func (s *AnalyticsService) IngestCampaignEvents(ctx context.Context, rows []port.CampaignEvent) error {
	if !s.enabled {
		return ErrDisabled
	}
	if s.store == nil {
		return ErrBackendUnavailable
	}
	if err := s.store.InsertCampaignEvents(ctx, rows); err != nil {
		return fmt.Errorf("ingest campaign events: %w", err)
	}

	return nil
}
