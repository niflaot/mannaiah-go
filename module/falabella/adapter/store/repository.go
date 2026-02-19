package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	syncdomain "mannaiah/module/falabella/domain/sync"
	"mannaiah/module/falabella/port"

	"gorm.io/gorm"
)

var (
	// ErrNilDB is returned when a nil DB dependency is provided.
	ErrNilDB = errors.New("falabella sync status db must not be nil")
)

// syncStatusRecord defines sync status persistence schema.
type syncStatusRecord struct {
	// FeedID defines Falabella feed identifier values (primary key).
	FeedID string `gorm:"primaryKey;size:191;not null"`
	// ProductID defines source product identifier values.
	ProductID string `gorm:"index;size:128;not null"`
	// SKU defines seller SKU values.
	SKU string `gorm:"size:128;not null"`
	// Action defines sync operation type values.
	Action string `gorm:"size:16;not null"`
	// Status defines feed resolution status values.
	Status string `gorm:"index;size:16;not null"`
	// SyncedAt defines sync submission timestamp values.
	SyncedAt time.Time `gorm:"not null"`
	// ResolvedAt defines optional feed resolution timestamp values.
	ResolvedAt *time.Time
}

// TableName defines storage table name.
func (syncStatusRecord) TableName() string {
	return "falabella_sync_status"
}

// Repository implements sync status persistence using GORM.
type Repository struct {
	// db is the underlying GORM handle.
	db *gorm.DB
}

var (
	// _ ensures Repository satisfies sync status repository contracts.
	_ port.SyncStatusRepository = (*Repository)(nil)
)

// NewRepository creates sync status repositories over GORM.
func NewRepository(db *gorm.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	return &Repository{db: db}, nil
}

// EnsureSchema migrates sync status persistence schema.
// Handles migration from old schema (id PK + feed_id unique) to new schema (feed_id PK).
func (r *Repository) EnsureSchema(ctx context.Context) error {
	migrator := r.db.WithContext(ctx).Migrator()

	if migrator.HasTable(&syncStatusRecord{}) && migrator.HasColumn(&syncStatusRecord{}, "id") {
		if err := migrator.DropTable(&syncStatusRecord{}); err != nil {
			return fmt.Errorf("drop legacy falabella sync status table: %w", err)
		}
	}

	if err := r.db.WithContext(ctx).AutoMigrate(&syncStatusRecord{}); err != nil {
		return fmt.Errorf("migrate falabella sync status schema: %w", err)
	}

	return nil
}

// Create persists a new sync status entry.
func (r *Repository) Create(ctx context.Context, entry *syncdomain.SyncEntry) error {
	record := toRecord(*entry)

	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return wrapWriteError("create", err)
	}

	return nil
}

// GetByFeedID retrieves a sync status entry by Falabella feed identifier.
func (r *Repository) GetByFeedID(ctx context.Context, feedID string) (*syncdomain.SyncEntry, error) {
	trimmedFeedID := strings.TrimSpace(feedID)

	var record syncStatusRecord
	if err := r.db.WithContext(ctx).First(&record, "feed_id = ?", trimmedFeedID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, port.ErrSyncEntryNotFound
		}
		return nil, fmt.Errorf("get sync status record: %w", err)
	}

	entity := toDomain(record)
	return &entity, nil
}

// GetByProductID retrieves sync status entries by source product identifier.
func (r *Repository) GetByProductID(ctx context.Context, productID string) ([]syncdomain.SyncEntry, error) {
	trimmedProductID := strings.TrimSpace(productID)

	var records []syncStatusRecord
	if err := r.db.WithContext(ctx).Where("product_id = ?", trimmedProductID).Order("synced_at DESC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list sync status records: %w", err)
	}

	entries := make([]syncdomain.SyncEntry, 0, len(records))
	for _, record := range records {
		entries = append(entries, toDomain(record))
	}

	return entries, nil
}

// ListPending retrieves unresolved sync status entries ordered by submission time.
func (r *Repository) ListPending(ctx context.Context, limit int) ([]syncdomain.SyncEntry, error) {
	resolvedLimit := limit
	if resolvedLimit <= 0 {
		resolvedLimit = 50
	}

	var records []syncStatusRecord
	if err := r.db.WithContext(ctx).
		Where("status = ?", string(syncdomain.SyncStatusPending)).
		Order("synced_at ASC").
		Limit(resolvedLimit).
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list pending sync status records: %w", err)
	}

	entries := make([]syncdomain.SyncEntry, 0, len(records))
	for _, record := range records {
		entries = append(entries, toDomain(record))
	}

	return entries, nil
}

// UpdateStatus updates the status and resolution timestamp of a sync status entry.
func (r *Repository) UpdateStatus(ctx context.Context, feedID string, status syncdomain.SyncStatus, resolvedAt *time.Time) error {
	trimmedFeedID := strings.TrimSpace(feedID)

	result := r.db.WithContext(ctx).Model(&syncStatusRecord{}).Where("feed_id = ?", trimmedFeedID).Updates(map[string]any{
		"status":      string(status),
		"resolved_at": resolvedAt,
	})
	if result.Error != nil {
		return fmt.Errorf("update sync status record: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return port.ErrSyncEntryNotFound
	}

	return nil
}

// toRecord maps domain sync entries to persistence records.
func toRecord(entry syncdomain.SyncEntry) syncStatusRecord {
	return syncStatusRecord{
		FeedID:     strings.TrimSpace(entry.FeedID),
		ProductID:  strings.TrimSpace(entry.ProductID),
		SKU:        strings.TrimSpace(entry.SKU),
		Action:     string(entry.Action),
		Status:     string(entry.Status),
		SyncedAt:   entry.SyncedAt,
		ResolvedAt: entry.ResolvedAt,
	}
}

// toDomain maps persistence records to domain sync entries.
func toDomain(record syncStatusRecord) syncdomain.SyncEntry {
	return syncdomain.SyncEntry{
		FeedID:     record.FeedID,
		ProductID:  record.ProductID,
		SKU:        record.SKU,
		Action:     syncdomain.SyncAction(record.Action),
		Status:     syncdomain.SyncStatus(record.Status),
		SyncedAt:   record.SyncedAt,
		ResolvedAt: record.ResolvedAt,
	}
}

// wrapWriteError normalizes persistence write errors to stable repository errors.
func wrapWriteError(operation string, err error) error {
	if mapDuplicateError(err) != nil {
		return fmt.Errorf("%s sync status record: %w", operation, port.ErrDuplicateFeedID)
	}

	return fmt.Errorf("%s sync status record: %w", operation, err)
}

// mapDuplicateError maps duplicate-key persistence failures.
func mapDuplicateError(err error) error {
	if err == nil {
		return nil
	}

	message := strings.ToLower(err.Error())
	if errors.Is(err, gorm.ErrDuplicatedKey) ||
		strings.Contains(message, "duplicate key") ||
		strings.Contains(message, "duplicated key") ||
		strings.Contains(message, "unique constraint failed") ||
		strings.Contains(message, "unique failed") {
		return port.ErrDuplicateFeedID
	}

	return nil
}