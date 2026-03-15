package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

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
		return Status{Enabled: true, BackendHealthy: false, Error: ErrBackendUnavailable.Error()}
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
	if s.store == nil {
		return nil, ErrBackendUnavailable
	}
	if err := s.store.EnsureSchema(ctx); err != nil {
		return nil, fmt.Errorf("ensure analytics schema: %w", err)
	}

	runID := ""
	if s.syncRecorder != nil {
		startedRunID, runErr := s.syncRecorder.StartRun(ctx, "analytics.seed", "manual")
		if runErr == nil {
			runID = startedRunID
		}
	}

	summary := &SeedSummary{}
	syncErrors := make([]port.SyncError, 0, 8)
	appendSyncError := func(errorType string, errorCode string, message string) {
		trimmedMessage := strings.TrimSpace(message)
		if trimmedMessage == "" {
			return
		}
		syncErrors = append(syncErrors, port.SyncError{Type: strings.TrimSpace(errorType), Code: strings.TrimSpace(errorCode), Message: trimmedMessage})
	}
	finalizeSyncRecord := func(failed bool) {
		if strings.TrimSpace(runID) == "" || s.syncRecorder == nil {
			return
		}
		processed := int(summary.Contacts + summary.Orders + summary.OrderItems + summary.MembershipEvents + summary.CampaignEvents)
		if failed {
			_ = s.syncRecorder.FailRun(ctx, runID, processed, processed, len(syncErrors), 0, syncErrors)
			return
		}
		_ = s.syncRecorder.CompleteRun(ctx, runID, processed, processed, 0, 0)
	}

	if err := s.seedContacts(ctx, summary); err != nil {
		appendSyncError("seed", "contacts_failed", err.Error())
		finalizeSyncRecord(true)
		return nil, err
	}
	if err := s.seedOrders(ctx, summary); err != nil {
		appendSyncError("seed", "orders_failed", err.Error())
		finalizeSyncRecord(true)
		return nil, err
	}
	if err := s.seedMembershipEvents(ctx, summary); err != nil {
		appendSyncError("seed", "membership_failed", err.Error())
		finalizeSyncRecord(true)
		return nil, err
	}
	if err := s.seedCampaignEvents(ctx, summary); err != nil {
		appendSyncError("seed", "campaign_failed", err.Error())
		finalizeSyncRecord(true)
		return nil, err
	}

	finalizeSyncRecord(false)
	return summary, nil
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

type contactSeedRow struct {
	ID           string     `gorm:"column:id"`
	Email        string     `gorm:"column:email"`
	FirstName    string     `gorm:"column:first_name"`
	LastName     string     `gorm:"column:last_name"`
	LegalName    string     `gorm:"column:legal_name"`
	Phone        string     `gorm:"column:phone"`
	CityCode     string     `gorm:"column:city_code"`
	DocumentType string     `gorm:"column:document_type"`
	CreatedAt    *time.Time `gorm:"column:created_at"`
	UpdatedAt    *time.Time `gorm:"column:updated_at"`
}

type contactMetadataSeedRow struct {
	ContactID string `gorm:"column:contact_id"`
	Key       string `gorm:"column:key"`
	Value     string `gorm:"column:value"`
}

func (s *AnalyticsService) seedContacts(ctx context.Context, summary *SeedSummary) error {
	lastID := ""
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		rows := make([]contactSeedRow, 0, seedBatchSize)
		query := s.db.WithContext(ctx).
			Table("contacts").
			Select("id", "email", "first_name", "last_name", "legal_name", "phone", "city_code", "document_type", "created_at", "updated_at").
			Where("deleted_at IS NULL")
		if lastID != "" {
			query = query.Where("id > ?", lastID)
		}
		if err := query.Order("id ASC").Limit(seedBatchSize).Scan(&rows).Error; err != nil {
			return fmt.Errorf("seed contacts batch: %w", err)
		}
		if len(rows) == 0 {
			break
		}

		ids := make([]string, 0, len(rows))
		for _, row := range rows {
			ids = append(ids, strings.TrimSpace(row.ID))
		}

		metadataRows := make([]contactMetadataSeedRow, 0, len(rows)*2)
		if err := s.db.WithContext(ctx).
			Table("contact_metadata").
			Select("contact_id, `key`, value").
			Where("contact_id IN ?", ids).
			Scan(&metadataRows).Error; err != nil {
			return fmt.Errorf("seed contact metadata batch: %w", err)
		}
		metadataByContact := map[string]map[string]string{}
		for _, row := range metadataRows {
			contactID := strings.TrimSpace(row.ContactID)
			if contactID == "" {
				continue
			}
			if _, exists := metadataByContact[contactID]; !exists {
				metadataByContact[contactID] = map[string]string{}
			}
			key := strings.TrimSpace(row.Key)
			if key == "" {
				continue
			}
			metadataByContact[contactID][key] = row.Value
		}

		payload := make([]port.ContactSnapshot, 0, len(rows))
		now := time.Now().UTC()
		for _, row := range rows {
			createdAt := now
			updatedAt := now
			if row.CreatedAt != nil {
				createdAt = row.CreatedAt.UTC()
			}
			if row.UpdatedAt != nil {
				updatedAt = row.UpdatedAt.UTC()
			}
			if createdAt.IsZero() {
				createdAt = now
			}
			if updatedAt.IsZero() {
				updatedAt = createdAt
			}

			contactID := strings.TrimSpace(row.ID)
			payload = append(payload, port.ContactSnapshot{
				ContactID:    contactID,
				Email:        strings.TrimSpace(row.Email),
				FirstName:    strings.TrimSpace(row.FirstName),
				LastName:     strings.TrimSpace(row.LastName),
				LegalName:    strings.TrimSpace(row.LegalName),
				Phone:        strings.TrimSpace(row.Phone),
				CityCode:     strings.TrimSpace(row.CityCode),
				DocumentType: strings.TrimSpace(row.DocumentType),
				Metadata:     metadataByContact[contactID],
				CreatedAt:    createdAt,
				UpdatedAt:    updatedAt,
			})
		}

		if err := s.store.UpsertContacts(ctx, payload); err != nil {
			return fmt.Errorf("upsert contacts snapshot batch: %w", err)
		}
		summary.Contacts += int64(len(payload))
		lastID = rows[len(rows)-1].ID
	}

	return nil
}

type orderSeedRow struct {
	ID         string     `gorm:"column:id"`
	Identifier string     `gorm:"column:identifier"`
	Realm      string     `gorm:"column:realm"`
	ContactID  string     `gorm:"column:contact_id"`
	CreatedAt  *time.Time `gorm:"column:created_at"`
	UpdatedAt  *time.Time `gorm:"column:updated_at"`
}

type orderItemSeedRow struct {
	OrderID          string  `gorm:"column:order_id"`
	SKU              string  `gorm:"column:sku"`
	AlternateName    string  `gorm:"column:alternate_name"`
	Quantity         int     `gorm:"column:quantity"`
	Value            float64 `gorm:"column:value"`
	ProductID        string  `gorm:"column:product_id"`
	ResolutionSource string  `gorm:"column:resolution_source"`
}

type orderStatusSeedRow struct {
	OrderID string `gorm:"column:order_id"`
	Status  string `gorm:"column:status"`
}

func (s *AnalyticsService) seedOrders(ctx context.Context, summary *SeedSummary) error {
	lastID := ""
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		orders := make([]orderSeedRow, 0, seedBatchSize)
		query := s.db.WithContext(ctx).
			Table("orders").
			Select("id", "identifier", "realm", "contact_id", "created_at", "updated_at").
			Where("deleted_at IS NULL")
		if lastID != "" {
			query = query.Where("id > ?", lastID)
		}
		if err := query.Order("id ASC").Limit(seedBatchSize).Scan(&orders).Error; err != nil {
			return fmt.Errorf("seed orders batch: %w", err)
		}
		if len(orders) == 0 {
			break
		}

		orderIDs := make([]string, 0, len(orders))
		orderByID := map[string]orderSeedRow{}
		now := time.Now().UTC()
		for _, row := range orders {
			orderID := strings.TrimSpace(row.ID)
			orderIDs = append(orderIDs, orderID)
			orderByID[orderID] = row
			if row.CreatedAt == nil {
				value := now
				row.CreatedAt = &value
			}
			if row.UpdatedAt == nil {
				value := now
				row.UpdatedAt = &value
			}
		}

		itemRows := make([]orderItemSeedRow, 0, len(orders)*2)
		if err := s.db.WithContext(ctx).
			Table("order_items").
			Select("order_id", "sku", "alternate_name", "quantity", "value", "product_id", "resolution_source").
			Where("order_id IN ?", orderIDs).
			Order("order_id ASC").
			Scan(&itemRows).Error; err != nil {
			return fmt.Errorf("seed order items batch: %w", err)
		}

		statusRows := make([]orderStatusSeedRow, 0, len(orders))
		statusQuery := `SELECT osh.order_id, osh.status
			FROM order_status_history osh
			INNER JOIN (
				SELECT order_id, max(occurred_at) AS latest_occurred_at
				FROM order_status_history
				WHERE order_id IN ?
				GROUP BY order_id
			) latest ON latest.order_id = osh.order_id AND latest.latest_occurred_at = osh.occurred_at`
		if err := s.db.WithContext(ctx).Raw(statusQuery, orderIDs).Scan(&statusRows).Error; err != nil {
			return fmt.Errorf("seed order statuses batch: %w", err)
		}
		statusByOrder := map[string]string{}
		for _, row := range statusRows {
			statusByOrder[strings.TrimSpace(row.OrderID)] = strings.TrimSpace(row.Status)
		}

		totalByOrder := map[string]float64{}
		countByOrder := map[string]int{}
		orderItemsPayload := make([]port.OrderItemFact, 0, len(itemRows))
		for _, row := range itemRows {
			orderID := strings.TrimSpace(row.OrderID)
			orderSeed, exists := orderByID[orderID]
			if !exists {
				continue
			}
			createdAt := now
			updatedAt := now
			if orderSeed.CreatedAt != nil {
				createdAt = orderSeed.CreatedAt.UTC()
			}
			if orderSeed.UpdatedAt != nil {
				updatedAt = orderSeed.UpdatedAt.UTC()
			}
			value := row.Value
			totalByOrder[orderID] += value
			countByOrder[orderID] += row.Quantity

			orderItemsPayload = append(orderItemsPayload, port.OrderItemFact{
				OrderID:          orderID,
				ContactID:        strings.TrimSpace(orderSeed.ContactID),
				SKU:              strings.TrimSpace(row.SKU),
				AlternateName:    strings.TrimSpace(row.AlternateName),
				ProductID:        strings.TrimSpace(row.ProductID),
				Quantity:         row.Quantity,
				Value:            value,
				ResolutionSource: strings.TrimSpace(row.ResolutionSource),
				OrderCreatedAt:   createdAt,
				OrderUpdatedAt:   updatedAt,
			})
		}

		ordersPayload := make([]port.OrderFact, 0, len(orders))
		for _, row := range orders {
			createdAt := now
			updatedAt := now
			if row.CreatedAt != nil {
				createdAt = row.CreatedAt.UTC()
			}
			if row.UpdatedAt != nil {
				updatedAt = row.UpdatedAt.UTC()
			}
			orderID := strings.TrimSpace(row.ID)
			ordersPayload = append(ordersPayload, port.OrderFact{
				OrderID:       orderID,
				Identifier:    strings.TrimSpace(row.Identifier),
				Realm:         strings.TrimSpace(row.Realm),
				ContactID:     strings.TrimSpace(row.ContactID),
				CurrentStatus: strings.TrimSpace(statusByOrder[orderID]),
				TotalValue:    totalByOrder[orderID],
				ItemCount:     countByOrder[orderID],
				CreatedAt:     createdAt,
				UpdatedAt:     updatedAt,
			})
		}

		if err := s.store.UpsertOrders(ctx, ordersPayload); err != nil {
			return fmt.Errorf("upsert orders fact batch: %w", err)
		}
		if err := s.store.UpsertOrderItems(ctx, orderItemsPayload); err != nil {
			return fmt.Errorf("upsert order items fact batch: %w", err)
		}
		summary.Orders += int64(len(ordersPayload))
		summary.OrderItems += int64(len(orderItemsPayload))

		lastID = orders[len(orders)-1].ID
	}

	return nil
}

type membershipSeedRow struct {
	ID         string    `gorm:"column:id"`
	ContactID  string    `gorm:"column:contact_id"`
	Channel    string    `gorm:"column:channel"`
	Action     string    `gorm:"column:action"`
	Source     string    `gorm:"column:source"`
	OccurredAt time.Time `gorm:"column:occurred_at"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

func (s *AnalyticsService) seedMembershipEvents(ctx context.Context, summary *SeedSummary) error {
	lastCreatedAt := time.Time{}
	lastID := ""
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		rows := make([]membershipSeedRow, 0, seedBatchSize)
		query := s.db.WithContext(ctx).
			Table("membership_stamps").
			Select("id", "contact_id", "channel", "action", "source", "occurred_at", "created_at")
		if !lastCreatedAt.IsZero() {
			query = query.Where("(created_at > ?) OR (created_at = ? AND id > ?)", lastCreatedAt, lastCreatedAt, lastID)
		}
		if err := query.Order("created_at ASC").Order("id ASC").Limit(seedBatchSize).Scan(&rows).Error; err != nil {
			return fmt.Errorf("seed membership events batch: %w", err)
		}
		if len(rows) == 0 {
			break
		}

		payload := make([]port.MembershipEvent, 0, len(rows))
		for _, row := range rows {
			payload = append(payload, port.MembershipEvent{
				ContactID:  strings.TrimSpace(row.ContactID),
				Channel:    strings.TrimSpace(row.Channel),
				Action:     strings.TrimSpace(row.Action),
				Source:     strings.TrimSpace(row.Source),
				OccurredAt: row.OccurredAt.UTC(),
			})
		}
		if err := s.store.InsertMembershipEvents(ctx, payload); err != nil {
			return fmt.Errorf("insert membership events batch: %w", err)
		}
		summary.MembershipEvents += int64(len(payload))
		lastCreatedAt = rows[len(rows)-1].CreatedAt.UTC()
		lastID = rows[len(rows)-1].ID
	}

	return nil
}

type campaignSeedRow struct {
	ID             string    `gorm:"column:id"`
	ContactID      string    `gorm:"column:contact_id"`
	IdempotencyKey string    `gorm:"column:idempotency_key"`
	Status         string    `gorm:"column:status"`
	OccurredAt     time.Time `gorm:"column:occurred_at"`
}

func (s *AnalyticsService) seedCampaignEvents(ctx context.Context, summary *SeedSummary) error {
	lastID := ""
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		rows := make([]campaignSeedRow, 0, seedBatchSize)
		query := s.db.WithContext(ctx).
			Table("email_delivery_status_history esh").
			Select("esh.id", "ed.contact_id", "ed.idempotency_key", "esh.status", "esh.occurred_at").
			Joins("INNER JOIN email_deliveries ed ON ed.id = esh.delivery_id")
		if lastID != "" {
			query = query.Where("esh.id > ?", lastID)
		}
		if err := query.Order("esh.id ASC").Limit(seedBatchSize).Scan(&rows).Error; err != nil {
			if looksLikeTableMissing(err) {
				return nil
			}
			return fmt.Errorf("seed campaign events batch: %w", err)
		}
		if len(rows) == 0 {
			break
		}

		payload := make([]port.CampaignEvent, 0, len(rows))
		for _, row := range rows {
			campaignID, fallbackContactID, ok := parseCampaignIdempotency(row.IdempotencyKey)
			if !ok {
				continue
			}
			contactID := strings.TrimSpace(row.ContactID)
			if contactID == "" {
				contactID = fallbackContactID
			}
			payload = append(payload, port.CampaignEvent{
				CampaignID:      campaignID,
				ContactID:       contactID,
				Channel:         "email",
				Status:          strings.TrimSpace(row.Status),
				TemplateVersion: 1,
				OccurredAt:      row.OccurredAt.UTC(),
			})
		}
		if err := s.store.InsertCampaignEvents(ctx, payload); err != nil {
			return fmt.Errorf("insert campaign events batch: %w", err)
		}
		summary.CampaignEvents += int64(len(payload))
		lastID = rows[len(rows)-1].ID
	}

	return nil
}

func parseCampaignIdempotency(value string) (string, string, bool) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) < 2 {
		return "", "", false
	}

	campaignID := strings.TrimSpace(parts[0])
	contactID := strings.TrimSpace(parts[1])
	if campaignID == "" || contactID == "" {
		return "", "", false
	}

	return campaignID, contactID, true
}

func looksLikeTableMissing(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	if strings.Contains(message, "doesn't exist") {
		return true
	}
	if strings.Contains(message, "no such table") {
		return true
	}

	return false
}
