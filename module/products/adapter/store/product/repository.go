package product

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	productdomain "mannaiah/module/products/domain/product"
	productport "mannaiah/module/products/port/product"

	"gorm.io/gorm"
)

var (
	// ErrNilDB is returned when DB dependencies are nil.
	ErrNilDB = errors.New("products db must not be nil")
)

// Repository implements product persistence using GORM.
type Repository struct {
	// db defines GORM dependencies.
	db *gorm.DB
}

// productRecord defines normalized product root persistence schema.
type productRecord struct {
	// ID defines unique identifiers.
	ID string `gorm:"primaryKey;size:64"`
	// SKU defines unique stock-keeping values.
	SKU string `gorm:"size:255;not null;uniqueIndex"`
	// Price defines optional product price values.
	Price *float64 `gorm:"column:price"`
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time
	// DeletedAt defines soft-delete timestamps.
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// productGalleryRecord defines normalized gallery item rows.
type productGalleryRecord struct {
	// ID defines surrogate identifiers.
	ID uint `gorm:"primaryKey"`
	// ProductID defines owning product identifiers.
	ProductID string `gorm:"size:64;not null;index"`
	// Position defines stable gallery ordering.
	Position int `gorm:"not null;index"`
	// VariationPosition defines optional variation-scoped gallery ordering.
	VariationPosition *int `gorm:"index"`
	// AssetID defines referenced asset identifiers.
	AssetID string `gorm:"size:64;not null"`
	// IsMain reports primary-image state.
	IsMain bool `gorm:"not null"`
}

// productGalleryIncludedRealmRecord defines included realm rows for gallery items.
type productGalleryIncludedRealmRecord struct {
	// ID defines surrogate identifiers.
	ID uint `gorm:"primaryKey"`
	// GalleryItemID defines owning gallery item identifiers.
	GalleryItemID uint `gorm:"not null;index"`
	// Position defines stable realm ordering.
	Position int `gorm:"not null;index"`
	// Realm defines included realm identifiers.
	Realm string `gorm:"size:128;not null"`
}

// productGalleryVariationRecord defines gallery item variation links.
type productGalleryVariationRecord struct {
	// ID defines surrogate identifiers.
	ID uint `gorm:"primaryKey"`
	// GalleryItemID defines owning gallery item identifiers.
	GalleryItemID uint `gorm:"not null;index"`
	// Position defines stable variation ordering.
	Position int `gorm:"not null;index"`
	// VariationID defines referenced variation identifiers.
	VariationID string `gorm:"size:64;not null"`
}

// productDatasheetRecord defines normalized datasheet rows.
type productDatasheetRecord struct {
	// ID defines surrogate identifiers.
	ID uint `gorm:"primaryKey"`
	// ProductID defines owning product identifiers.
	ProductID string `gorm:"size:64;not null;index"`
	// Position defines stable datasheet ordering.
	Position int `gorm:"not null;index"`
	// Realm defines target realm identifiers.
	Realm string `gorm:"size:128;not null"`
	// Name defines display names.
	Name string `gorm:"size:255;not null"`
	// Description defines optional descriptions.
	Description string `gorm:"type:text"`
}

// productDatasheetAttributeRecord defines datasheet attribute rows.
type productDatasheetAttributeRecord struct {
	// ID defines surrogate identifiers.
	ID uint `gorm:"primaryKey"`
	// DatasheetID defines owning datasheet identifiers.
	DatasheetID uint `gorm:"not null;index"`
	// Key defines attribute keys.
	Key string `gorm:"size:128;not null"`
	// ValueJSON defines JSON-encoded attribute values.
	ValueJSON string `gorm:"type:text;not null"`
}

// productVariationLinkRecord defines product-level variation links.
type productVariationLinkRecord struct {
	// ID defines surrogate identifiers.
	ID uint `gorm:"primaryKey"`
	// ProductID defines owning product identifiers.
	ProductID string `gorm:"size:64;not null;index"`
	// Position defines stable variation ordering.
	Position int `gorm:"not null;index"`
	// VariationID defines referenced variation identifiers.
	VariationID string `gorm:"size:64;not null"`
}

// productVariantRecord defines product variant rows.
type productVariantRecord struct {
	// ID defines surrogate identifiers.
	ID uint `gorm:"primaryKey"`
	// ProductID defines owning product identifiers.
	ProductID string `gorm:"size:64;not null;index"`
	// Position defines stable variant ordering.
	Position int `gorm:"not null;index"`
	// SKU defines variant-level SKU values.
	SKU string `gorm:"size:255"`
}

// productVariantVariationRecord defines variation links per variant.
type productVariantVariationRecord struct {
	// ID defines surrogate identifiers.
	ID uint `gorm:"primaryKey"`
	// VariantID defines owning variant identifiers.
	VariantID uint `gorm:"not null;index"`
	// Position defines stable variation ordering.
	Position int `gorm:"not null;index"`
	// VariationID defines referenced variation identifiers.
	VariationID string `gorm:"size:64;not null"`
}

// TableName defines storage table name.
func (productRecord) TableName() string { return "products" }

// TableName defines storage table name.
func (productGalleryRecord) TableName() string { return "product_gallery_items" }

// TableName defines storage table name.
func (productGalleryIncludedRealmRecord) TableName() string { return "product_gallery_included_realms" }

// TableName defines storage table name.
func (productGalleryVariationRecord) TableName() string { return "product_gallery_variations" }

// TableName defines storage table name.
func (productDatasheetRecord) TableName() string { return "product_datasheets" }

// TableName defines storage table name.
func (productDatasheetAttributeRecord) TableName() string { return "product_datasheet_attributes" }

// TableName defines storage table name.
func (productVariationLinkRecord) TableName() string { return "product_variation_links" }

// TableName defines storage table name.
func (productVariantRecord) TableName() string { return "product_variants" }

// TableName defines storage table name.
func (productVariantVariationRecord) TableName() string { return "product_variant_variations" }

var (
	// _ ensures repository contract compliance.
	_ productport.Repository = (*Repository)(nil)
)

// NewRepository creates product repositories.
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

// Create persists product entities.
func (r *Repository) Create(ctx context.Context, entity *productdomain.Product) error {
	record := productRecord{ID: strings.TrimSpace(entity.ID), SKU: strings.TrimSpace(entity.SKU), Price: entity.Price}
	if record.ID == "" {
		record.ID = generateID()
	}

	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&record).Error; err != nil {
			if isDuplicateSKUErr(err) {
				return productport.ErrDuplicateSKU
			}
			return fmt.Errorf("create product record: %w", err)
		}
		if err := replaceProductRelations(tx, record.ID, *entity); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	loaded, err := r.GetByID(ctx, record.ID)
	if err != nil {
		return err
	}
	*entity = *loaded

	return nil
}

// GetByID retrieves products by ID.
func (r *Repository) GetByID(ctx context.Context, id string) (*productdomain.Product, error) {
	var record productRecord
	if err := r.db.WithContext(ctx).First(&record, "id = ?", strings.TrimSpace(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, productport.ErrNotFound
		}
		return nil, fmt.Errorf("get product record: %w", err)
	}

	entity, err := r.loadProductAggregate(ctx, record)
	if err != nil {
		return nil, err
	}

	return &entity, nil
}

// GetBySKU retrieves products by product-level or variant-level SKU.
func (r *Repository) GetBySKU(ctx context.Context, sku string) (*productdomain.Product, error) {
	trimmed := strings.TrimSpace(sku)

	var record productRecord
	err := r.db.WithContext(ctx).First(&record, "sku = ?", trimmed).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("get product by sku: %w", err)
	}

	if err == nil {
		entity, loadErr := r.loadProductAggregate(ctx, record)
		if loadErr != nil {
			return nil, loadErr
		}
		return &entity, nil
	}

	var productID string
	variantErr := r.db.WithContext(ctx).
		Model(&productVariantRecord{}).
		Select("product_id").
		Where("sku = ?", trimmed).
		Order("id ASC").
		Limit(1).
		Pluck("product_id", &productID).Error
	if variantErr != nil {
		return nil, fmt.Errorf("get product by variant sku: %w", variantErr)
	}
	if productID == "" {
		return nil, productport.ErrNotFound
	}

	return r.GetByID(ctx, productID)
}

// List retrieves all non-deleted products.
func (r *Repository) List(ctx context.Context) ([]productdomain.Product, error) {
	records := make([]productRecord, 0)
	if err := r.db.WithContext(ctx).Order("created_at desc").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list product records: %w", err)
	}

	result := make([]productdomain.Product, 0, len(records))
	for _, record := range records {
		entity, err := r.loadProductAggregate(ctx, record)
		if err != nil {
			return nil, err
		}
		result = append(result, entity)
	}

	return result, nil
}

// Update persists product updates.
func (r *Repository) Update(ctx context.Context, entity *productdomain.Product) error {
	productID := strings.TrimSpace(entity.ID)
	if productID == "" {
		return productport.ErrNotFound
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{"sku": strings.TrimSpace(entity.SKU), "price": entity.Price}
		updateTx := tx.Model(&productRecord{}).Where("id = ?", productID).Updates(updates)
		if updateTx.Error != nil {
			if isDuplicateSKUErr(updateTx.Error) {
				return productport.ErrDuplicateSKU
			}
			return fmt.Errorf("update product record: %w", updateTx.Error)
		}
		if updateTx.RowsAffected == 0 {
			return productport.ErrNotFound
		}

		if err := replaceProductRelations(tx, productID, *entity); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	latest, err := r.GetByID(ctx, productID)
	if err != nil {
		return err
	}
	*entity = *latest

	return nil
}

// Delete soft-deletes products by ID.
func (r *Repository) Delete(ctx context.Context, id string) error {
	trimmedID := strings.TrimSpace(id)

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		deleteTx := tx.Delete(&productRecord{}, "id = ?", trimmedID)
		if deleteTx.Error != nil {
			return fmt.Errorf("delete product record: %w", deleteTx.Error)
		}
		if deleteTx.RowsAffected == 0 {
			return productport.ErrNotFound
		}
		if err := clearProductRelations(tx, trimmedID); err != nil {
			return err
		}

		return nil
	})
}

// GetByIDs retrieves multiple products by their IDs.
func (r *Repository) GetByIDs(ctx context.Context, ids []string) ([]*productdomain.Product, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	records := make([]productRecord, 0)
	if err := r.db.WithContext(ctx).Where("id IN ? AND deleted_at IS NULL", ids).Find(&records).Error; err != nil {
		return nil, fmt.Errorf("get products by ids: %w", err)
	}

	result := make([]*productdomain.Product, 0, len(records))
	for _, record := range records {
		entity, err := r.loadProductAggregate(ctx, record)
		if err != nil {
			return nil, err
		}
		e := entity
		result = append(result, &e)
	}

	return result, nil
}

// ListByTagsAndPrice retrieves products matching tag and price criteria with pagination.
func (r *Repository) ListByTagsAndPrice(ctx context.Context, tags []string, minPrice, maxPrice *float64, page, pageSize int) ([]*productdomain.Product, int64, error) {
	query := r.db.WithContext(ctx).Model(&productRecord{}).Where("deleted_at IS NULL")

	if len(tags) > 0 {
		query = query.Where("id IN (?)",
			r.db.Model(&productTagRecord{}).Select("product_id").Where("tag IN ?", tags),
		)
	}
	if minPrice != nil {
		query = query.Where("price >= ?", *minPrice)
	}
	if maxPrice != nil {
		query = query.Where("price <= ?", *maxPrice)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count products by tags and price: %w", err)
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	records := make([]productRecord, 0)
	if err := query.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, 0, fmt.Errorf("list products by tags and price: %w", err)
	}

	result := make([]*productdomain.Product, 0, len(records))
	for _, record := range records {
		entity, err := r.loadProductAggregate(ctx, record)
		if err != nil {
			return nil, 0, err
		}
		e := entity
		result = append(result, &e)
	}

	return result, total, nil
}

// isDuplicateSKUErr reports SKU-unique-constraint violations.
func isDuplicateSKUErr(err error) bool {
	if err == nil {
		return false
	}

	value := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(value, "unique") && strings.Contains(value, "sku")
}

// generateID creates product identifiers.
func generateID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}

	return hex.EncodeToString(bytes)
}
