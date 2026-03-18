package tag

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	tagdomain "mannaiah/module/products/domain/tag"
	tagport "mannaiah/module/products/port/tag"
)

var (
	// ErrNilDB is returned when DB dependencies are nil.
	ErrNilDB = errors.New("tags db must not be nil")
)

// tagRecord defines the canonical tag registry row.
type tagRecord struct {
	// ID defines surrogate identifiers.
	ID uint `gorm:"primaryKey"`
	// Name defines unique tag name values.
	Name string `gorm:"size:128;not null;uniqueIndex"`
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time
	// DeletedAt defines optional soft-delete timestamps.
	DeletedAt *time.Time `gorm:"index"`
}

// tagCorrelationRecord defines a cross-sell probability mapping row.
type tagCorrelationRecord struct {
	// ID defines surrogate identifiers.
	ID uint `gorm:"primaryKey"`
	// SourceTag defines the source tag name.
	SourceTag string `gorm:"size:128;not null;uniqueIndex:idx_tag_correlations_pair"`
	// TargetTag defines the correlated target tag name.
	TargetTag string `gorm:"size:128;not null;uniqueIndex:idx_tag_correlations_pair"`
	// Probability defines cross-sell purchase probability (0.00–100.00).
	Probability float64 `gorm:"column:probability;not null;default:0.00"`
	// Notes defines optional marketing notes.
	Notes string `gorm:"type:text"`
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time
}

// TableName defines storage table name.
func (tagRecord) TableName() string { return "tags" }

// TableName defines storage table name.
func (tagCorrelationRecord) TableName() string { return "tag_correlations" }

// Repository implements tag persistence using GORM.
type Repository struct {
	// db defines GORM dependencies.
	db *gorm.DB
}

var (
	// _ ensures repository contract compliance.
	_ tagport.Repository = (*Repository)(nil)
)

// NewRepository creates tag repositories.
func NewRepository(db *gorm.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	return &Repository{db: db}, nil
}

// EnsureAll creates missing tags and reintegrates soft-deleted ones.
// For each name: creates a new tag row if absent, or clears deleted_at if soft-deleted.
func (r *Repository) EnsureAll(ctx context.Context, names []string) error {
	for _, name := range names {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}

		var existing tagRecord
		err := r.db.WithContext(ctx).Where("name = ?", trimmed).First(&existing).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("find tag %q: %w", trimmed, err)
		}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			if createErr := r.db.WithContext(ctx).Create(&tagRecord{Name: trimmed}).Error; createErr != nil {
				return fmt.Errorf("create tag %q: %w", trimmed, createErr)
			}
			continue
		}

		if existing.DeletedAt != nil {
			if execErr := r.db.WithContext(ctx).Exec(
				"UPDATE tags SET deleted_at = NULL, updated_at = NOW(3) WHERE id = ?",
				existing.ID,
			).Error; execErr != nil {
				return fmt.Errorf("reintegrate tag %q: %w", trimmed, execErr)
			}
		}
	}

	return nil
}

// List returns all non-deleted tags ordered by name.
func (r *Repository) List(ctx context.Context) ([]tagdomain.Tag, error) {
	var records []tagRecord
	if err := r.db.WithContext(ctx).Where("deleted_at IS NULL").Order("name ASC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}

	tags := make([]tagdomain.Tag, 0, len(records))
	for _, rec := range records {
		tags = append(tags, toTagDomain(rec))
	}

	return tags, nil
}

// SoftDelete soft-deletes a tag by name and cascades removal to product_tags.
func (r *Repository) SoftDelete(ctx context.Context, name string) error {
	trimmed := strings.TrimSpace(name)

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Exec(
			"UPDATE tags SET deleted_at = NOW(3), updated_at = NOW(3) WHERE name = ? AND deleted_at IS NULL",
			trimmed,
		)
		if result.Error != nil {
			return fmt.Errorf("soft-delete tag: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return tagport.ErrNotFound
		}

		if err := tx.Exec("DELETE FROM product_tags WHERE tag = ?", trimmed).Error; err != nil {
			return fmt.Errorf("cascade delete product tags: %w", err)
		}

		return nil
	})
}

// ListCorrelations returns all correlation records ordered by source tag.
func (r *Repository) ListCorrelations(ctx context.Context) ([]tagdomain.TagCorrelation, error) {
	var records []tagCorrelationRecord
	if err := r.db.WithContext(ctx).Order("source_tag ASC, target_tag ASC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list correlations: %w", err)
	}

	return toCorrelationDomains(records), nil
}

// ListCorrelationsBySource returns correlations for a specific source tag.
func (r *Repository) ListCorrelationsBySource(ctx context.Context, sourceTag string) ([]tagdomain.TagCorrelation, error) {
	trimmed := strings.TrimSpace(sourceTag)

	var records []tagCorrelationRecord
	if err := r.db.WithContext(ctx).Where("source_tag = ?", trimmed).Order("target_tag ASC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list correlations by source: %w", err)
	}

	return toCorrelationDomains(records), nil
}

// CreateCorrelation persists a new correlation and populates ID and timestamps on success.
func (r *Repository) CreateCorrelation(ctx context.Context, correlation *tagdomain.TagCorrelation) error {
	record := tagCorrelationRecord{
		SourceTag:   strings.TrimSpace(correlation.SourceTag),
		TargetTag:   strings.TrimSpace(correlation.TargetTag),
		Probability: correlation.Probability,
		Notes:       strings.TrimSpace(correlation.Notes),
	}

	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		if isDuplicateCorrelationErr(err) {
			return tagport.ErrDuplicateCorrelation
		}
		return fmt.Errorf("create correlation: %w", err)
	}

	correlation.ID = record.ID
	correlation.CreatedAt = record.CreatedAt
	correlation.UpdatedAt = record.UpdatedAt

	return nil
}

// UpdateCorrelation updates correlation fields by ID and returns the refreshed record.
func (r *Repository) UpdateCorrelation(ctx context.Context, id uint, probability *float64, notes *string) (*tagdomain.TagCorrelation, error) {
	updates := map[string]any{}
	if probability != nil {
		updates["probability"] = *probability
	}
	if notes != nil {
		updates["notes"] = *notes
	}

	if len(updates) == 0 {
		var record tagCorrelationRecord
		if err := r.db.WithContext(ctx).First(&record, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, tagport.ErrCorrelationNotFound
			}
			return nil, fmt.Errorf("get correlation: %w", err)
		}
		result := toCorrelationDomain(record)
		return &result, nil
	}

	updateResult := r.db.WithContext(ctx).Model(&tagCorrelationRecord{}).Where("id = ?", id).Updates(updates)
	if updateResult.Error != nil {
		return nil, fmt.Errorf("update correlation: %w", updateResult.Error)
	}
	if updateResult.RowsAffected == 0 {
		return nil, tagport.ErrCorrelationNotFound
	}

	var record tagCorrelationRecord
	if err := r.db.WithContext(ctx).First(&record, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("reload correlation: %w", err)
	}

	updated := toCorrelationDomain(record)
	return &updated, nil
}

// DeleteCorrelation hard-deletes a correlation record by ID.
func (r *Repository) DeleteCorrelation(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&tagCorrelationRecord{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete correlation: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return tagport.ErrCorrelationNotFound
	}

	return nil
}

// toTagDomain converts a tagRecord to the domain Tag type.
func toTagDomain(rec tagRecord) tagdomain.Tag {
	return tagdomain.Tag{
		ID:        rec.ID,
		Name:      rec.Name,
		CreatedAt: rec.CreatedAt,
		UpdatedAt: rec.UpdatedAt,
		DeletedAt: rec.DeletedAt,
	}
}

// toCorrelationDomain converts a tagCorrelationRecord to the domain TagCorrelation type.
func toCorrelationDomain(rec tagCorrelationRecord) tagdomain.TagCorrelation {
	return tagdomain.TagCorrelation{
		ID:          rec.ID,
		SourceTag:   rec.SourceTag,
		TargetTag:   rec.TargetTag,
		Probability: rec.Probability,
		Notes:       rec.Notes,
		CreatedAt:   rec.CreatedAt,
		UpdatedAt:   rec.UpdatedAt,
	}
}

// toCorrelationDomains converts a slice of tagCorrelationRecord to domain types.
func toCorrelationDomains(records []tagCorrelationRecord) []tagdomain.TagCorrelation {
	result := make([]tagdomain.TagCorrelation, 0, len(records))
	for _, rec := range records {
		result = append(result, toCorrelationDomain(rec))
	}
	return result
}

// isDuplicateCorrelationErr reports unique-pair constraint violations.
func isDuplicateCorrelationErr(err error) bool {
	if err == nil {
		return false
	}
	value := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(value, "unique") && strings.Contains(value, "idx_tag_correlations_pair")
}
