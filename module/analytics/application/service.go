package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"mannaiah/module/analytics/domain"
	"mannaiah/module/analytics/port"
)

var (
	// ErrNilDB is returned when nil db dependencies are provided.
	ErrNilDB = errors.New("analytics db must not be nil")
	// ErrDisabled is returned when analytics are disabled by configuration.
	ErrDisabled = errors.New("analytics module is disabled")
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
}

// Service defines analytics use-case behavior.
type Service interface {
	// Status returns analytics runtime health values.
	Status(ctx context.Context) Status
	// Seed executes best-effort initial data seeding behavior.
	Seed(ctx context.Context) (*SeedSummary, error)
	// ResolveContacts resolves contact ids for analytical filters.
	ResolveContacts(ctx context.Context, filter domain.SegmentFilter, page int, limit int) ([]string, error)
}

// AnalyticsService implements analytics use-cases.
type AnalyticsService struct {
	// enabled defines analytics enablement state.
	enabled bool
	// db defines transactional database dependencies used for read fallback and seed counts.
	db *gorm.DB
	// store defines optional analytics backend dependencies.
	store port.Store
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
		return Status{Enabled: true, BackendHealthy: true}
	}

	if err := s.store.Ping(ctx); err != nil {
		return Status{Enabled: true, BackendHealthy: false, Error: err.Error()}
	}

	return Status{Enabled: true, BackendHealthy: true}
}

// Seed executes best-effort initial data seeding behavior.
func (s *AnalyticsService) Seed(ctx context.Context) (*SeedSummary, error) {
	if !s.enabled {
		return nil, ErrDisabled
	}

	runID := ""
	if s.syncRecorder != nil {
		startedRunID, runErr := s.syncRecorder.StartRun(ctx, "analytics.seed", "manual")
		if runErr == nil {
			runID = startedRunID
		}
	}

	failRun := func(message string, processed int) {
		if strings.TrimSpace(runID) == "" || s.syncRecorder == nil {
			return
		}
		_ = s.syncRecorder.FailRun(ctx, runID, processed, processed, 1, 0, []port.SyncError{{
			Type:    "seed",
			Code:    "count_failed",
			Message: strings.TrimSpace(message),
		}})
	}

	summary := &SeedSummary{}
	if err := s.db.WithContext(ctx).Table("contacts").Count(&summary.Contacts).Error; err != nil {
		failRun(err.Error(), 0)
		return nil, fmt.Errorf("count contacts for analytics seed: %w", err)
	}
	if err := s.db.WithContext(ctx).Table("orders").Count(&summary.Orders).Error; err != nil {
		processed := int(summary.Contacts)
		failRun(err.Error(), processed)
		return nil, fmt.Errorf("count orders for analytics seed: %w", err)
	}
	if strings.TrimSpace(runID) != "" && s.syncRecorder != nil {
		processed := int(summary.Contacts + summary.Orders)
		_ = s.syncRecorder.CompleteRun(ctx, runID, processed, processed, 0, 0)
	}

	return summary, nil
}

// ResolveContacts resolves contact ids for analytical filters.
func (s *AnalyticsService) ResolveContacts(ctx context.Context, filter domain.SegmentFilter, page int, limit int) ([]string, error) {
	if !s.enabled {
		return nil, ErrDisabled
	}
	if limit <= 0 {
		limit = 1000
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	db := s.db.WithContext(ctx).Table("contacts c").Select("c.id").Where("c.deleted_at IS NULL")
	if len(filter.CityCodes) > 0 {
		db = db.Where("c.city_code IN ?", filter.CityCodes)
	}
	if filter.RequireEmailOptIn {
		db = db.Where("EXISTS (SELECT 1 FROM membership_status ms WHERE ms.contact_id = c.id AND ms.channel = ? AND ms.action = ?)", "email", "opt_in")
	}
	if filter.MinTotalSpend != nil {
		db = db.Where("EXISTS (SELECT 1 FROM orders o WHERE o.contact_id = c.id GROUP BY o.contact_id HAVING SUM(o.total_value) >= ?)", *filter.MinTotalSpend)
	}
	if strings.TrimSpace(filter.PurchasedSKU) != "" {
		db = db.Where("EXISTS (SELECT 1 FROM orders o JOIN order_items oi ON oi.order_id = o.id WHERE o.contact_id = c.id AND oi.sku = ?)", strings.TrimSpace(filter.PurchasedSKU))
	}

	ids := make([]string, 0, limit)
	if err := db.Order("c.id ASC").Offset(offset).Limit(limit).Scan(&ids).Error; err != nil {
		return nil, fmt.Errorf("resolve analytics contacts: %w", err)
	}

	return ids, nil
}
