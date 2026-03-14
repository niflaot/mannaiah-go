package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"mannaiah/module/syncrecord/domain"
	"mannaiah/module/syncrecord/port"
)

var (
	// ErrNilDB is returned when nil db dependencies are provided.
	ErrNilDB = errors.New("sync record db must not be nil")
)

// Repository defines GORM-backed sync record persistence behavior.
type Repository struct {
	// db defines GORM database dependencies.
	db *gorm.DB
}

var (
	// _ ensures Repository satisfies sync record repository contracts.
	_ port.Repository = (*Repository)(nil)
)

// NewRepository creates GORM-backed sync record repositories.
func NewRepository(db *gorm.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	return &Repository{db: db}, nil
}

// CreateRun persists a new running sync run.
func (r *Repository) CreateRun(ctx context.Context, run *domain.SyncRun) error {
	model := mapRunToModel(run)
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return fmt.Errorf("insert sync run: %w", err)
	}

	return nil
}

// CompleteRun updates run status and counters with terminal values.
func (r *Repository) CompleteRun(ctx context.Context, input port.CompleteInput) error {
	trimmedRunID := strings.TrimSpace(input.RunID)
	if trimmedRunID == "" {
		return domain.ErrInvalidRunID
	}

	endedAt := input.EndedAt.UTC()
	updates := map[string]any{
		"status":          strings.TrimSpace(string(input.Status)),
		"ended_at":        endedAt,
		"duration_ms":     0,
		"processed_count": input.Processed,
		"succeeded_count": input.Succeeded,
		"failed_count":    input.Failed,
		"skipped_count":   input.Skipped,
		"updated_at":      endedAt,
	}

	result := r.db.WithContext(ctx).
		Model(&runModel{}).
		Where("id = ? AND status = ?", trimmedRunID, string(domain.RunStatusRunning)).
		Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("update sync run: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		existing := runModel{}
		err := r.db.WithContext(ctx).Where("id = ?", trimmedRunID).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrRunNotFound
		}
		if err != nil {
			return fmt.Errorf("load sync run: %w", err)
		}

		return domain.ErrRunAlreadyFinished
	}

	return r.recalculateDurationAndErrorCount(ctx, trimmedRunID)
}

// AddRunErrors persists child error rows for a run.
func (r *Repository) AddRunErrors(ctx context.Context, entries []domain.SyncRunError) error {
	if len(entries) == 0 {
		return nil
	}

	models := make([]runErrorModel, 0, len(entries))
	for _, entry := range entries {
		models = append(models, mapErrorToModel(entry))
	}

	if err := r.db.WithContext(ctx).Create(&models).Error; err != nil {
		return fmt.Errorf("insert sync run errors: %w", err)
	}

	return nil
}

// GetRunByID retrieves a run with child errors by id.
func (r *Repository) GetRunByID(ctx context.Context, runID string) (*domain.SyncRun, error) {
	trimmedRunID := strings.TrimSpace(runID)
	if trimmedRunID == "" {
		return nil, domain.ErrInvalidRunID
	}

	model := runModel{}
	err := r.db.WithContext(ctx).
		Preload("Errors", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("created_at ASC")
		}).
		Where("id = ?", trimmedRunID).
		First(&model).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrRunNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select sync run by id: %w", err)
	}

	run := mapModelToRun(model)
	return &run, nil
}

// ListRuns returns paged run rows and total count for filters.
func (r *Repository) ListRuns(ctx context.Context, query port.ListQuery) ([]domain.SyncRun, int64, error) {
	db := r.db.WithContext(ctx).Model(&runModel{})
	db = applyListFilters(db, query)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count sync runs: %w", err)
	}
	if total == 0 {
		return []domain.SyncRun{}, 0, nil
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}
	page := query.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	models := make([]runModel, 0, limit)
	if err := db.Preload("Errors", func(tx *gorm.DB) *gorm.DB {
		return tx.Order("created_at ASC")
	}).Order("started_at DESC").Offset(offset).Limit(limit).Find(&models).Error; err != nil {
		return nil, 0, fmt.Errorf("list sync runs: %w", err)
	}

	runs := make([]domain.SyncRun, 0, len(models))
	for _, model := range models {
		runs = append(runs, mapModelToRun(model))
	}

	return runs, total, nil
}

// StatsSince returns aggregate stats from a lower-bound timestamp.
func (r *Repository) StatsSince(ctx context.Context, since time.Time) (*domain.RunStats, error) {
	type row struct {
		TotalRuns     int64      `gorm:"column:total_runs"`
		CompletedRuns int64      `gorm:"column:completed_runs"`
		FailedRuns    int64      `gorm:"column:failed_runs"`
		AvgDurationMS int64      `gorm:"column:avg_duration_ms"`
		LastFailureAt *time.Time `gorm:"column:last_failure_at"`
	}

	result := row{}
	err := r.db.WithContext(ctx).Model(&runModel{}).
		Select(
			"COUNT(*) AS total_runs",
			"SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) AS completed_runs",
			"SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) AS failed_runs",
			"COALESCE(AVG(duration_ms), 0) AS avg_duration_ms",
			"MAX(CASE WHEN status = ? THEN ended_at ELSE NULL END) AS last_failure_at",
			string(domain.RunStatusCompleted),
			string(domain.RunStatusFailed),
			string(domain.RunStatusFailed),
		).
		Where("started_at >= ?", since.UTC()).
		Scan(&result).
		Error
	if err != nil {
		return nil, fmt.Errorf("select sync run stats: %w", err)
	}

	stats := &domain.RunStats{
		WindowStart:   since.UTC(),
		TotalRuns:     result.TotalRuns,
		CompletedRuns: result.CompletedRuns,
		FailedRuns:    result.FailedRuns,
		AvgDurationMS: result.AvgDurationMS,
		LastFailureAt: result.LastFailureAt,
	}
	return stats, nil
}

// CleanupBefore deletes runs older than cutoff and returns deleted rows count.
func (r *Repository) CleanupBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("started_at < ?", cutoff.UTC()).
		Delete(&runModel{})
	if result.Error != nil {
		return 0, fmt.Errorf("delete old sync runs: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// applyListFilters applies list-query filters to base queries.
func applyListFilters(db *gorm.DB, query port.ListQuery) *gorm.DB {
	if db == nil {
		return db
	}

	kind := strings.TrimSpace(query.Kind)
	if kind != "" {
		db = db.Where("kind = ?", kind)
	}
	trigger := strings.TrimSpace(query.Trigger)
	if trigger != "" {
		db = db.Where("trigger = ?", trigger)
	}
	status := strings.TrimSpace(query.Status)
	if status != "" {
		db = db.Where("status = ?", status)
	}
	if query.StartedAfter != nil && !query.StartedAfter.IsZero() {
		db = db.Where("started_at >= ?", query.StartedAfter.UTC())
	}
	if query.StartedBefore != nil && !query.StartedBefore.IsZero() {
		db = db.Where("started_at <= ?", query.StartedBefore.UTC())
	}

	return db
}

// recalculateDurationAndErrorCount updates derived counters for one run.
func (r *Repository) recalculateDurationAndErrorCount(ctx context.Context, runID string) error {
	type row struct {
		StartedAt time.Time  `gorm:"column:started_at"`
		EndedAt   *time.Time `gorm:"column:ended_at"`
	}
	current := row{}
	if err := r.db.WithContext(ctx).Model(&runModel{}).Select("started_at", "ended_at").Where("id = ?", runID).Scan(&current).Error; err != nil {
		return fmt.Errorf("load sync run timing: %w", err)
	}

	errorCount := int64(0)
	if err := r.db.WithContext(ctx).Model(&runErrorModel{}).Where("run_id = ?", runID).Count(&errorCount).Error; err != nil {
		return fmt.Errorf("count sync run errors: %w", err)
	}

	durationMS := int64(0)
	if current.EndedAt != nil && !current.StartedAt.IsZero() {
		delta := current.EndedAt.UTC().Sub(current.StartedAt.UTC())
		if delta > 0 {
			durationMS = delta.Milliseconds()
		}
	}

	if err := r.db.WithContext(ctx).Model(&runModel{}).Where("id = ?", runID).Updates(map[string]any{
		"duration_ms": durationMS,
		"error_count": errorCount,
	}).Error; err != nil {
		return fmt.Errorf("update sync run derived counters: %w", err)
	}

	return nil
}
