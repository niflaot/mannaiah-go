package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"mannaiah/module/assets/domain"
	"mannaiah/module/assets/port"
)

var (
	// ErrNilDB is returned when a nil DB dependency is provided.
	ErrNilDB = errors.New("assets db must not be nil")
)

// Repository implements asset persistence using GORM.
type Repository struct {
	// db is the underlying GORM handle.
	db *gorm.DB
}

// assetRecord defines persistence schema for assets.
type assetRecord struct {
	// ID defines primary key identifiers.
	ID string `gorm:"primaryKey;size:64"`
	// Key defines storage object key paths.
	Key string `gorm:"uniqueIndex:idx_assets_key;size:512;not null"`
	// Name defines custom display names.
	Name string `gorm:"size:255;not null"`
	// OriginalName defines original uploaded file names.
	OriginalName string `gorm:"size:255;not null"`
	// MimeType defines object mime type values.
	MimeType string `gorm:"size:255;not null"`
	// Size defines object size in bytes.
	Size int64 `gorm:"not null"`
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time
	// DeletedAt defines soft-delete timestamps.
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName defines storage table names.
func (assetRecord) TableName() string {
	return "assets"
}

var (
	// _ ensures Repository satisfies asset repository contracts.
	_ port.Repository = (*Repository)(nil)
)

// NewRepository creates an asset repository over GORM.
func NewRepository(db *gorm.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	return &Repository{db: db}, nil
}

// EnsureSchema migrates asset persistence schema.
func (r *Repository) EnsureSchema(ctx context.Context) error {
	if err := r.db.WithContext(ctx).AutoMigrate(&assetRecord{}); err != nil {
		return fmt.Errorf("migrate asset schema: %w", err)
	}

	return nil
}

// Create persists asset metadata rows.
func (r *Repository) Create(ctx context.Context, asset *domain.Asset) error {
	record := toRecord(*asset)
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return fmt.Errorf("create asset record: %w", err)
	}

	*asset = toDomain(record)
	return nil
}

// GetByID retrieves asset rows by id.
func (r *Repository) GetByID(ctx context.Context, id string) (*domain.Asset, error) {
	var record assetRecord
	if err := r.db.WithContext(ctx).First(&record, "id = ?", strings.TrimSpace(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, port.ErrNotFound
		}

		return nil, fmt.Errorf("get asset record: %w", err)
	}

	entity := toDomain(record)
	return &entity, nil
}

// List paginates asset metadata rows.
func (r *Repository) List(ctx context.Context, query port.ListQuery) (*port.PageResult, error) {
	page, limit := normalizePagination(query.Page, query.Limit)

	base := r.db.WithContext(ctx).Model(&assetRecord{})
	base = applyListFilters(base, query.Filters)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count asset records: %w", err)
	}

	records := make([]assetRecord, 0)
	if err := base.Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list asset records: %w", err)
	}

	result := make([]domain.Asset, 0, len(records))
	for _, record := range records {
		result = append(result, toDomain(record))
	}

	return &port.PageResult{
		Data:  result,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

// UpdateName updates asset custom names.
func (r *Repository) UpdateName(ctx context.Context, id string, name string) (*domain.Asset, error) {
	trimmedID := strings.TrimSpace(id)
	trimmedName := strings.TrimSpace(name)

	tx := r.db.WithContext(ctx).Model(&assetRecord{}).Where("id = ?", trimmedID).Update("name", trimmedName)
	if tx.Error != nil {
		return nil, fmt.Errorf("update asset name: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return nil, port.ErrNotFound
	}

	return r.GetByID(ctx, trimmedID)
}

// SoftDelete soft-deletes asset metadata rows.
func (r *Repository) SoftDelete(ctx context.Context, id string) error {
	tx := r.db.WithContext(ctx).Delete(&assetRecord{}, "id = ?", strings.TrimSpace(id))
	if tx.Error != nil {
		return fmt.Errorf("soft delete asset record: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return port.ErrNotFound
	}

	return nil
}

// toRecord maps domain entities into persistence records.
func toRecord(asset domain.Asset) assetRecord {
	return assetRecord{
		ID:           strings.TrimSpace(asset.ID),
		Key:          strings.TrimSpace(asset.Key),
		Name:         strings.TrimSpace(asset.Name),
		OriginalName: strings.TrimSpace(asset.OriginalName),
		MimeType:     strings.TrimSpace(asset.MimeType),
		Size:         asset.Size,
		CreatedAt:    asset.CreatedAt,
		UpdatedAt:    asset.UpdatedAt,
	}
}

// toDomain maps persistence records into domain entities.
func toDomain(record assetRecord) domain.Asset {
	return domain.Asset{
		ID:           record.ID,
		Key:          record.Key,
		Name:         record.Name,
		OriginalName: record.OriginalName,
		MimeType:     record.MimeType,
		Size:         record.Size,
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
		IsDeleted:    record.DeletedAt.Valid,
		DeletedAt:    toDeletedAtPointer(record.DeletedAt),
	}
}

// toDeletedAtPointer maps gorm deleted-at values into pointer timestamps.
func toDeletedAtPointer(deletedAt gorm.DeletedAt) *time.Time {
	if !deletedAt.Valid {
		return nil
	}

	value := deletedAt.Time
	return &value
}

// applyListFilters applies list search filters across relevant columns.
func applyListFilters(tx *gorm.DB, filters string) *gorm.DB {
	trimmed := strings.TrimSpace(filters)
	if trimmed == "" {
		return tx
	}

	pattern := "%" + strings.ToLower(trimmed) + "%"
	return tx.Where(
		"LOWER(name) LIKE ? OR LOWER(original_name) LIKE ? OR LOWER(mime_type) LIKE ? OR LOWER(`key`) LIKE ?",
		pattern,
		pattern,
		pattern,
		pattern,
	)
}

// normalizePagination resolves list pagination defaults.
func normalizePagination(page int, limit int) (int, int) {
	resolvedPage := page
	if resolvedPage <= 0 {
		resolvedPage = 1
	}

	resolvedLimit := limit
	if resolvedLimit <= 0 {
		resolvedLimit = 10
	}
	if resolvedLimit > 100 {
		resolvedLimit = 100
	}

	return resolvedPage, resolvedLimit
}
