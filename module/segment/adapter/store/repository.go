package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"mannaiah/module/segment/domain"
	"mannaiah/module/segment/port"
)

var (
	// ErrNilDB is returned when nil db dependencies are provided.
	ErrNilDB = errors.New("segment db must not be nil")
)

// model defines segment persistence row values.
type model struct {
	// ID defines segment identifier values.
	ID string `gorm:"column:id;primaryKey"`
	// Name defines human-readable segment names.
	Name string `gorm:"column:name"`
	// Slug defines URL-safe segment slugs.
	Slug string `gorm:"column:slug"`
	// Channel defines channel values.
	Channel string `gorm:"column:channel"`
	// ParentSegmentID defines optional parent segment references.
	ParentSegmentID *string `gorm:"column:parent_segment_id"`
	// FiltersJSON defines filter payload values serialized as JSON.
	FiltersJSON string `gorm:"column:filters_json"`
	// CreatedAt defines row creation timestamp values.
	CreatedAt time.Time `gorm:"column:created_at"`
	// UpdatedAt defines row update timestamp values.
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

// TableName resolves segment table names.
func (model) TableName() string {
	return "segments"
}

// Repository defines GORM-backed segment persistence behavior.
type Repository struct {
	// db defines GORM database dependencies.
	db *gorm.DB
}

var (
	// _ ensures Repository satisfies segment repository contracts.
	_ port.Repository = (*Repository)(nil)
)

// NewRepository creates GORM-backed segment repositories.
func NewRepository(db *gorm.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	return &Repository{db: db}, nil
}

// Create persists segment rows.
func (r *Repository) Create(ctx context.Context, segment *domain.Segment) error {
	if segment.ID == "" {
		segment.ID = uuid.NewString()
	}
	row := model{
		ID:              segment.ID,
		Name:            segment.Name,
		Slug:            segment.Slug,
		Channel:         segment.Channel,
		ParentSegmentID: segment.ParentSegmentID,
		FiltersJSON:     marshalFilters(segment.Filters),
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return fmt.Errorf("create segment: %w", err)
	}

	segment.CreatedAt = row.CreatedAt.UTC()
	segment.UpdatedAt = row.UpdatedAt.UTC()
	return nil
}

// GetByID retrieves one segment by id.
func (r *Repository) GetByID(ctx context.Context, id string) (*domain.Segment, error) {
	row := model{}
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select segment by id: %w", err)
	}

	return mapModel(row), nil
}

// List retrieves segment rows.
func (r *Repository) List(ctx context.Context, page int, limit int) ([]domain.Segment, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit

	var total int64
	if err := r.db.WithContext(ctx).Model(&model{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count segments: %w", err)
	}

	rows := make([]model, 0, limit)
	if err := r.db.WithContext(ctx).Order("created_at DESC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list segments: %w", err)
	}

	segments := make([]domain.Segment, 0, len(rows))
	for _, row := range rows {
		segments = append(segments, *mapModel(row))
	}

	return segments, total, nil
}

// Update persists segment row updates.
func (r *Repository) Update(ctx context.Context, segment *domain.Segment) error {
	updates := map[string]any{
		"name":              segment.Name,
		"slug":              segment.Slug,
		"channel":           segment.Channel,
		"parent_segment_id": segment.ParentSegmentID,
		"filters_json":      marshalFilters(segment.Filters),
	}
	result := r.db.WithContext(ctx).Model(&model{}).Where("id = ?", segment.ID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("update segment: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	updated, err := r.GetByID(ctx, segment.ID)
	if err != nil {
		return err
	}
	segment.CreatedAt = updated.CreatedAt
	segment.UpdatedAt = updated.UpdatedAt
	return nil
}

// Delete removes one segment by id.
func (r *Repository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&model{})
	if result.Error != nil {
		return fmt.Errorf("delete segment: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// mapModel maps persistence rows into domain values.
func mapModel(row model) *domain.Segment {
	return &domain.Segment{
		ID:              row.ID,
		Name:            row.Name,
		Slug:            row.Slug,
		Channel:         row.Channel,
		ParentSegmentID: row.ParentSegmentID,
		Filters:         unmarshalFilters(row.FiltersJSON),
		CreatedAt:       row.CreatedAt.UTC(),
		UpdatedAt:       row.UpdatedAt.UTC(),
	}
}

// marshalFilters serializes filter payload values.
func marshalFilters(filters []domain.Filter) string {
	if len(filters) == 0 {
		return "[]"
	}
	payload, err := json.Marshal(filters)
	if err != nil {
		return "[]"
	}

	return string(payload)
}

// unmarshalFilters deserializes filter payload values.
func unmarshalFilters(value string) []domain.Filter {
	filters := make([]domain.Filter, 0)
	if err := json.Unmarshal([]byte(value), &filters); err != nil {
		return []domain.Filter{}
	}

	return filters
}
