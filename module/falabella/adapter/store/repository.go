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
	"gorm.io/gorm/clause"
)

var (
	// ErrNilDB is returned when a nil DB dependency is provided.
	ErrNilDB = errors.New("falabella sync status db must not be nil")
)

// syncStatusRecord defines sync status persistence schema.
type syncStatusRecord struct {
	// ExecutionID defines parent sync execution identifier values.
	ExecutionID string `gorm:"index;size:191;not null"`
	// FeedID defines Falabella feed identifier values (primary key).
	FeedID string `gorm:"primaryKey;size:191;not null"`
	// ProductID defines source product identifier values.
	ProductID string `gorm:"index;size:128;not null"`
	// SKU defines seller SKU values.
	SKU string `gorm:"size:128;not null"`
	// Step defines logical step values for this feed (product/image).
	Step string `gorm:"size:16;not null"`
	// Action defines sync operation type values.
	Action string `gorm:"size:16;not null"`
	// Status defines feed resolution status values.
	Status string `gorm:"index;size:16;not null"`
	// SyncedAt defines sync submission timestamp values.
	SyncedAt time.Time `gorm:"not null"`
	// ResolvedAt defines optional feed resolution timestamp values.
	ResolvedAt *time.Time
}

// syncExecutionRecord defines parent sync execution persistence schema.
type syncExecutionRecord struct {
	// ExecutionID defines one unique sync execution identifier.
	ExecutionID string `gorm:"primaryKey;size:191;not null"`
	// StartedAt defines sync execution start timestamp values.
	StartedAt time.Time `gorm:"not null"`
}

// TableName defines parent execution table name.
func (syncExecutionRecord) TableName() string {
	return "falabella_sync_execution"
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

// EnsureSchema is a no-op because schema evolution is managed by SQL migrations.
func (r *Repository) EnsureSchema(ctx context.Context) error {
	_ = ctx

	return nil
}

// CreateExecution persists one sync execution parent record.
func (r *Repository) CreateExecution(ctx context.Context, execution *syncdomain.SyncExecution) error {
	if execution == nil {
		return errors.New("sync execution must not be nil")
	}

	record := syncExecutionRecord{
		ExecutionID: strings.TrimSpace(execution.ExecutionID),
		StartedAt:   execution.StartedAt,
	}
	if record.ExecutionID == "" {
		return errors.New("sync execution id must not be empty")
	}
	if record.StartedAt.IsZero() {
		record.StartedAt = time.Now().UTC()
	}

	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&record).Error; err != nil {
		return fmt.Errorf("create sync execution: %w", err)
	}

	return nil
}

// Create persists a new sync status entry.
func (r *Repository) Create(ctx context.Context, entry *syncdomain.SyncEntry) error {
	if entry == nil {
		return errors.New("sync entry must not be nil")
	}

	record := toRecord(*entry)
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		execution := syncExecutionRecord{ExecutionID: record.ExecutionID, StartedAt: record.SyncedAt}
		if execution.ExecutionID == "" {
			return errors.New("sync execution id must not be empty")
		}
		if execution.StartedAt.IsZero() {
			execution.StartedAt = time.Now().UTC()
		}

		if createExecutionErr := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&execution).Error; createExecutionErr != nil {
			return fmt.Errorf("create sync execution: %w", createExecutionErr)
		}

		if createEntryErr := tx.Create(&record).Error; createEntryErr != nil {
			return wrapWriteError("create", createEntryErr)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// GetExecutionByID retrieves one sync execution by identifier.
func (r *Repository) GetExecutionByID(ctx context.Context, executionID string) (*syncdomain.SyncExecution, error) {
	trimmedExecutionID := strings.TrimSpace(executionID)

	var record syncExecutionRecord
	if err := r.db.WithContext(ctx).First(&record, "execution_id = ?", trimmedExecutionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, port.ErrSyncExecutionNotFound
		}
		return nil, fmt.Errorf("get sync execution record: %w", err)
	}

	execution := syncdomain.SyncExecution{ExecutionID: record.ExecutionID, StartedAt: record.StartedAt}
	return &execution, nil
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

// ListByExecutionID retrieves child feed rows by execution identifier ordered by submission time.
func (r *Repository) ListByExecutionID(ctx context.Context, executionID string) ([]syncdomain.SyncEntry, error) {
	trimmedExecutionID := strings.TrimSpace(executionID)

	var records []syncStatusRecord
	if err := r.db.WithContext(ctx).Where("execution_id = ?", trimmedExecutionID).Order("synced_at ASC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list sync status records by execution: %w", err)
	}

	entries := make([]syncdomain.SyncEntry, 0, len(records))
	for _, record := range records {
		entries = append(entries, toDomain(record))
	}

	return entries, nil
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
	executionID := strings.TrimSpace(entry.ExecutionID)
	if executionID == "" {
		executionID = strings.TrimSpace(entry.FeedID)
	}
	step := entry.Step
	if !step.IsValid() {
		step = syncdomain.SyncStepProduct
	}
	syncedAt := entry.SyncedAt
	if syncedAt.IsZero() {
		syncedAt = time.Now().UTC()
	}

	return syncStatusRecord{
		ExecutionID: executionID,
		FeedID:      strings.TrimSpace(entry.FeedID),
		ProductID:   strings.TrimSpace(entry.ProductID),
		SKU:         strings.TrimSpace(entry.SKU),
		Step:        step.String(),
		Action:      string(entry.Action),
		Status:      string(entry.Status),
		SyncedAt:    syncedAt,
		ResolvedAt:  entry.ResolvedAt,
	}
}

// toDomain maps persistence records to domain sync entries.
func toDomain(record syncStatusRecord) syncdomain.SyncEntry {
	return syncdomain.SyncEntry{
		ExecutionID: record.ExecutionID,
		FeedID:      record.FeedID,
		ProductID:   record.ProductID,
		SKU:         record.SKU,
		Step:        syncdomain.SyncStep(record.Step),
		Action:      syncdomain.SyncAction(record.Action),
		Status:      syncdomain.SyncStatus(record.Status),
		SyncedAt:    record.SyncedAt,
		ResolvedAt:  record.ResolvedAt,
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
