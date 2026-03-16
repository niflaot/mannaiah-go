package category

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	categorydomain "mannaiah/module/products/domain/category"
	categoryport "mannaiah/module/products/port/category"

	"gorm.io/gorm"
)

var (
	// ErrNilDB is returned when the DB dependency is nil.
	ErrNilDB = errors.New("category db must not be nil")
)

// categoryRecord defines normalized category root persistence schema.
type categoryRecord struct {
	// ID defines unique category identifiers.
	ID string `gorm:"primaryKey;size:64"`
	// Slug defines URL-friendly unique category slugs.
	Slug string `gorm:"size:255;not null;uniqueIndex"`
	// Name defines human-readable category names.
	Name string `gorm:"size:255;not null"`
	// Description defines optional category descriptions.
	Description string `gorm:"type:text"`
	// ParentID defines optional parent category identifiers.
	ParentID *string `gorm:"size:64;index"`
	// IncludeChildren reports whether descendant categories are included.
	IncludeChildren bool `gorm:"not null;default:false"`
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time
	// DeletedAt defines soft-delete timestamps.
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// categoryFilterTagRecord defines category filter tag rows.
type categoryFilterTagRecord struct {
	// ID defines surrogate identifiers.
	ID uint `gorm:"primaryKey"`
	// CategoryID defines owning category identifiers.
	CategoryID string `gorm:"size:64;not null;index"`
	// Position defines stable tag ordering.
	Position int `gorm:"not null"`
	// Tag defines tag values.
	Tag string `gorm:"size:128;not null"`
}

// categoryFilterPriceRangeRecord defines category price range filter rows.
type categoryFilterPriceRangeRecord struct {
	// ID defines surrogate identifiers.
	ID uint `gorm:"primaryKey"`
	// CategoryID defines owning category identifiers.
	CategoryID string `gorm:"size:64;not null;uniqueIndex"`
	// MinPrice defines optional minimum price values.
	MinPrice *float64
	// MaxPrice defines optional maximum price values.
	MaxPrice *float64
}

// categoryFilterCategoryRefRecord defines category-ref filter rows.
type categoryFilterCategoryRefRecord struct {
	// ID defines surrogate identifiers.
	ID uint `gorm:"primaryKey"`
	// CategoryID defines owning category identifiers.
	CategoryID string `gorm:"size:64;not null;index"`
	// RefCategoryID defines referenced category identifiers.
	RefCategoryID string `gorm:"size:64;not null"`
}

// categoryProductRecord defines manually pinned product rows per category.
type categoryProductRecord struct {
	// ID defines surrogate identifiers.
	ID uint `gorm:"primaryKey"`
	// CategoryID defines owning category identifiers.
	CategoryID string `gorm:"size:64;not null;index"`
	// ProductID defines pinned product identifiers.
	ProductID string `gorm:"size:64;not null"`
	// Position defines stable product ordering.
	Position int `gorm:"not null;default:0"`
}

// TableName defines storage table name.
func (categoryRecord) TableName() string { return "categories" }

// TableName defines storage table name.
func (categoryFilterTagRecord) TableName() string { return "category_filter_tags" }

// TableName defines storage table name.
func (categoryFilterPriceRangeRecord) TableName() string { return "category_filter_price_ranges" }

// TableName defines storage table name.
func (categoryFilterCategoryRefRecord) TableName() string { return "category_filter_category_refs" }

// TableName defines storage table name.
func (categoryProductRecord) TableName() string { return "category_products" }

var (
	// _ ensures Repository satisfies the port contract.
	_ categoryport.Repository = (*Repository)(nil)
)

// Repository implements category persistence using GORM.
type Repository struct {
	// db defines GORM database dependencies.
	db *gorm.DB
}

// NewRepository creates category repositories.
func NewRepository(db *gorm.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	return &Repository{db: db}, nil
}

// EnsureSchema is a no-op because schema evolution is managed by SQL migrations.
func (r *Repository) EnsureSchema(_ context.Context) error { return nil }

// Create persists a new category.
func (r *Repository) Create(ctx context.Context, cat *categorydomain.Category) error {
	record := toRecord(cat)
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&record).Error; err != nil {
			if isDuplicateSlugErr(err) {
				return categoryport.ErrDuplicateSlug
			}

			return fmt.Errorf("create category record: %w", err)
		}

		return replaceRelations(tx, cat.ID, cat)
	})
}

// GetByID retrieves a category by ID.
func (r *Repository) GetByID(ctx context.Context, id string) (*categorydomain.Category, error) {
	var record categoryRecord
	err := r.db.WithContext(ctx).First(&record, "id = ? AND deleted_at IS NULL", strings.TrimSpace(id)).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, categoryport.ErrNotFound
		}

		return nil, fmt.Errorf("get category record: %w", err)
	}

	return r.loadAggregate(ctx, record)
}

// GetBySlug retrieves a category by slug.
func (r *Repository) GetBySlug(ctx context.Context, slug string) (*categorydomain.Category, error) {
	var record categoryRecord
	err := r.db.WithContext(ctx).First(&record, "slug = ? AND deleted_at IS NULL", strings.TrimSpace(slug)).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, categoryport.ErrNotFound
		}

		return nil, fmt.Errorf("get category by slug: %w", err)
	}

	return r.loadAggregate(ctx, record)
}

// Tree retrieves all root-level non-deleted categories.
func (r *Repository) Tree(ctx context.Context) ([]*categorydomain.Category, error) {
	var records []categoryRecord
	if err := r.db.WithContext(ctx).Where("parent_id IS NULL AND deleted_at IS NULL").Order("created_at asc").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list category tree: %w", err)
	}

	return r.loadAggregates(ctx, records)
}

// ListChildren retrieves direct children of a category.
func (r *Repository) ListChildren(ctx context.Context, parentID string) ([]*categorydomain.Category, error) {
	var records []categoryRecord
	err := r.db.WithContext(ctx).Where("parent_id = ? AND deleted_at IS NULL", strings.TrimSpace(parentID)).Order("created_at asc").Find(&records).Error
	if err != nil {
		return nil, fmt.Errorf("list category children: %w", err)
	}

	return r.loadAggregates(ctx, records)
}

// Update persists category updates.
func (r *Repository) Update(ctx context.Context, cat *categorydomain.Category) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{
			"slug":             cat.Slug,
			"name":             cat.Name,
			"description":      cat.Description,
			"parent_id":        cat.ParentID,
			"include_children": cat.IncludeChildren,
			"updated_at":       cat.UpdatedAt,
		}
		res := tx.Model(&categoryRecord{}).Where("id = ? AND deleted_at IS NULL", cat.ID).Updates(updates)
		if res.Error != nil {
			if isDuplicateSlugErr(res.Error) {
				return categoryport.ErrDuplicateSlug
			}

			return fmt.Errorf("update category record: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return categoryport.ErrNotFound
		}

		return replaceRelations(tx, cat.ID, cat)
	})
}

// Delete soft-deletes a category by ID.
func (r *Repository) Delete(ctx context.Context, id string) error {
	trimmedID := strings.TrimSpace(id)

	var childCount int64
	if err := r.db.WithContext(ctx).Model(&categoryRecord{}).Where("parent_id = ? AND deleted_at IS NULL", trimmedID).Count(&childCount).Error; err != nil {
		return fmt.Errorf("count category children: %w", err)
	}
	if childCount > 0 {
		return categoryport.ErrHasChildren
	}

	res := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", trimmedID).Delete(&categoryRecord{})
	if res.Error != nil {
		return fmt.Errorf("delete category record: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return categoryport.ErrNotFound
	}

	return nil
}

// loadAggregate loads a full category aggregate from a root record.
func (r *Repository) loadAggregate(ctx context.Context, record categoryRecord) (*categorydomain.Category, error) {
	cat := &categorydomain.Category{
		ID:              record.ID,
		Slug:            record.Slug,
		Name:            record.Name,
		Description:     record.Description,
		ParentID:        record.ParentID,
		IncludeChildren: record.IncludeChildren,
		CreatedAt:       record.CreatedAt,
		UpdatedAt:       record.UpdatedAt,
	}

	tagRows := make([]categoryFilterTagRecord, 0)
	if err := r.db.WithContext(ctx).Where("category_id = ?", record.ID).Order("position asc").Find(&tagRows).Error; err != nil {
		return nil, fmt.Errorf("load category filter tags: %w", err)
	}
	for _, row := range tagRows {
		cat.Filter.Tags = append(cat.Filter.Tags, row.Tag)
	}

	var priceRow categoryFilterPriceRangeRecord
	err := r.db.WithContext(ctx).Where("category_id = ?", record.ID).First(&priceRow).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("load category price range: %w", err)
	}
	if err == nil {
		cat.Filter.PriceRange = &categorydomain.PriceRange{Min: priceRow.MinPrice, Max: priceRow.MaxPrice}
	}

	refRows := make([]categoryFilterCategoryRefRecord, 0)
	if err := r.db.WithContext(ctx).Where("category_id = ?", record.ID).Find(&refRows).Error; err != nil {
		return nil, fmt.Errorf("load category ref filters: %w", err)
	}
	for _, row := range refRows {
		cat.Filter.CategoryRefs = append(cat.Filter.CategoryRefs, row.RefCategoryID)
	}

	productRows := make([]categoryProductRecord, 0)
	if err := r.db.WithContext(ctx).Where("category_id = ?", record.ID).Order("position asc").Find(&productRows).Error; err != nil {
		return nil, fmt.Errorf("load category products: %w", err)
	}
	for _, row := range productRows {
		cat.ProductIDs = append(cat.ProductIDs, row.ProductID)
	}

	return cat, nil
}

// loadAggregates loads multiple category aggregates from root records.
func (r *Repository) loadAggregates(ctx context.Context, records []categoryRecord) ([]*categorydomain.Category, error) {
	result := make([]*categorydomain.Category, 0, len(records))
	for _, record := range records {
		cat, err := r.loadAggregate(ctx, record)
		if err != nil {
			return nil, err
		}
		result = append(result, cat)
	}

	return result, nil
}

// replaceRelations replaces all category relation rows from aggregate state.
func replaceRelations(tx *gorm.DB, categoryID string, cat *categorydomain.Category) error {
	if err := tx.Where("category_id = ?", categoryID).Delete(&categoryFilterTagRecord{}).Error; err != nil {
		return fmt.Errorf("delete category filter tags: %w", err)
	}
	for i, tag := range cat.Filter.Tags {
		if strings.TrimSpace(tag) == "" {
			continue
		}
		if err := tx.Create(&categoryFilterTagRecord{CategoryID: categoryID, Position: i, Tag: strings.TrimSpace(tag)}).Error; err != nil {
			return fmt.Errorf("create category filter tag: %w", err)
		}
	}

	if err := tx.Where("category_id = ?", categoryID).Delete(&categoryFilterPriceRangeRecord{}).Error; err != nil {
		return fmt.Errorf("delete category price range: %w", err)
	}
	if cat.Filter.PriceRange != nil {
		pr := categoryFilterPriceRangeRecord{CategoryID: categoryID, MinPrice: cat.Filter.PriceRange.Min, MaxPrice: cat.Filter.PriceRange.Max}
		if err := tx.Create(&pr).Error; err != nil {
			return fmt.Errorf("create category price range: %w", err)
		}
	}

	if err := tx.Where("category_id = ?", categoryID).Delete(&categoryFilterCategoryRefRecord{}).Error; err != nil {
		return fmt.Errorf("delete category ref filters: %w", err)
	}
	for _, refID := range cat.Filter.CategoryRefs {
		if strings.TrimSpace(refID) == "" {
			continue
		}
		if err := tx.Create(&categoryFilterCategoryRefRecord{CategoryID: categoryID, RefCategoryID: strings.TrimSpace(refID)}).Error; err != nil {
			return fmt.Errorf("create category ref filter: %w", err)
		}
	}

	if err := tx.Where("category_id = ?", categoryID).Delete(&categoryProductRecord{}).Error; err != nil {
		return fmt.Errorf("delete category products: %w", err)
	}
	for i, productID := range cat.ProductIDs {
		if strings.TrimSpace(productID) == "" {
			continue
		}
		if err := tx.Create(&categoryProductRecord{CategoryID: categoryID, ProductID: strings.TrimSpace(productID), Position: i}).Error; err != nil {
			return fmt.Errorf("create category product: %w", err)
		}
	}

	return nil
}

// toRecord converts a category domain entity to a persistence record.
func toRecord(cat *categorydomain.Category) categoryRecord {
	return categoryRecord{
		ID:              cat.ID,
		Slug:            cat.Slug,
		Name:            cat.Name,
		Description:     cat.Description,
		ParentID:        cat.ParentID,
		IncludeChildren: cat.IncludeChildren,
		CreatedAt:       cat.CreatedAt,
		UpdatedAt:       cat.UpdatedAt,
	}
}

// isDuplicateSlugErr reports slug unique-constraint violations.
func isDuplicateSlugErr(err error) bool {
	if err == nil {
		return false
	}
	val := strings.ToLower(strings.TrimSpace(err.Error()))

	return strings.Contains(val, "unique") && strings.Contains(val, "slug")
}
