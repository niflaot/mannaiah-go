package store

import (
	"context"
	"errors"
	"fmt"
	"sort"
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

// syncStatusVariationRecord defines sync status variation-link persistence schema.
type syncStatusVariationRecord struct {
	// FeedID defines Falabella feed identifier values.
	FeedID string `gorm:"primaryKey;size:191;not null"`
	// VariationID defines linked product variation identifier values.
	VariationID string `gorm:"primaryKey;size:128;not null"`
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

// TableName defines storage table name.
func (syncStatusVariationRecord) TableName() string {
	return "falabella_sync_status_variation"
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
	variationRecords := toVariationRecords(record.FeedID, entry.VariationIDs)
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
		if len(variationRecords) > 0 {
			if createVariationErr := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&variationRecords).Error; createVariationErr != nil {
				return fmt.Errorf("create sync status variation records: %w", createVariationErr)
			}
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

	variationIDs, err := r.listVariationIDsByFeedIDs(ctx, []string{record.FeedID})
	if err != nil {
		return nil, err
	}

	entity := toDomain(record, variationIDs[record.FeedID])
	return &entity, nil
}

// ListByExecutionID retrieves child feed rows by execution identifier ordered by submission time.
func (r *Repository) ListByExecutionID(ctx context.Context, executionID string) ([]syncdomain.SyncEntry, error) {
	trimmedExecutionID := strings.TrimSpace(executionID)

	var records []syncStatusRecord
	if err := r.db.WithContext(ctx).Where("execution_id = ?", trimmedExecutionID).Order("synced_at ASC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list sync status records by execution: %w", err)
	}

	return r.toDomainEntriesWithVariations(ctx, records)
}

// GetByProductID retrieves sync status entries by source product identifier.
func (r *Repository) GetByProductID(ctx context.Context, productID string) ([]syncdomain.SyncEntry, error) {
	trimmedProductID := strings.TrimSpace(productID)

	var records []syncStatusRecord
	if err := r.db.WithContext(ctx).Where("product_id = ?", trimmedProductID).Order("synced_at DESC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list sync status records: %w", err)
	}

	return r.toDomainEntriesWithVariations(ctx, records)
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

	return r.toDomainEntriesWithVariations(ctx, records)
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
func toDomain(record syncStatusRecord, variationIDs []string) syncdomain.SyncEntry {
	return syncdomain.SyncEntry{
		ExecutionID:  record.ExecutionID,
		FeedID:       record.FeedID,
		ProductID:    record.ProductID,
		SKU:          record.SKU,
		VariationIDs: append([]string(nil), variationIDs...),
		Step:         syncdomain.SyncStep(record.Step),
		Action:       syncdomain.SyncAction(record.Action),
		Status:       syncdomain.SyncStatus(record.Status),
		SyncedAt:     record.SyncedAt,
		ResolvedAt:   record.ResolvedAt,
	}
}

// toVariationRecords maps feed-linked variation identifiers into persistence records.
func toVariationRecords(feedID string, variationIDs []string) []syncStatusVariationRecord {
	trimmedFeedID := strings.TrimSpace(feedID)
	if trimmedFeedID == "" {
		return nil
	}

	normalizedVariationIDs := normalizeVariationIDs(variationIDs)
	if len(normalizedVariationIDs) == 0 {
		return nil
	}

	records := make([]syncStatusVariationRecord, 0, len(normalizedVariationIDs))
	for _, variationID := range normalizedVariationIDs {
		records = append(records, syncStatusVariationRecord{
			FeedID:      trimmedFeedID,
			VariationID: variationID,
		})
	}

	return records
}

// toDomainEntriesWithVariations maps status records into domain entries and resolves linked variation identifiers.
func (r *Repository) toDomainEntriesWithVariations(ctx context.Context, records []syncStatusRecord) ([]syncdomain.SyncEntry, error) {
	entries := make([]syncdomain.SyncEntry, 0, len(records))
	if len(records) == 0 {
		return entries, nil
	}

	feedIDs := make([]string, 0, len(records))
	for _, record := range records {
		feedIDs = append(feedIDs, record.FeedID)
	}

	variationIDsByFeedID, err := r.listVariationIDsByFeedIDs(ctx, feedIDs)
	if err != nil {
		return nil, err
	}

	for _, record := range records {
		entries = append(entries, toDomain(record, variationIDsByFeedID[record.FeedID]))
	}

	return entries, nil
}

// listVariationIDsByFeedIDs resolves linked variation identifiers grouped by feed identifier values.
func (r *Repository) listVariationIDsByFeedIDs(ctx context.Context, feedIDs []string) (map[string][]string, error) {
	result := map[string][]string{}
	if len(feedIDs) == 0 {
		return result, nil
	}

	normalizedFeedIDs := normalizeVariationIDs(feedIDs)
	if len(normalizedFeedIDs) == 0 {
		return result, nil
	}

	var records []syncStatusVariationRecord
	if err := r.db.WithContext(ctx).
		Where("feed_id IN ?", normalizedFeedIDs).
		Order("variation_id ASC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list sync status variation records: %w", err)
	}

	for _, record := range records {
		result[record.FeedID] = append(result[record.FeedID], record.VariationID)
	}

	return result, nil
}

// normalizeVariationIDs resolves sorted, deduplicated, trimmed identifier values.
func normalizeVariationIDs(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return nil
	}

	sort.Strings(normalized)
	return normalized
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
