package product

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	productdomain "mannaiah/module/products/domain/product"
	productport "mannaiah/module/products/port/product"
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

// productRecord defines product persistence schema.
type productRecord struct {
	// ID defines unique identifiers.
	ID string `gorm:"primaryKey;size:64"`
	// SKU defines unique stock-keeping values.
	SKU string `gorm:"size:255;not null;uniqueIndex"`
	// Gallery defines encoded gallery JSON payload.
	Gallery string `gorm:"type:text;not null"`
	// Datasheets defines encoded datasheet JSON payload.
	Datasheets string `gorm:"type:text;not null"`
	// Variations defines encoded variation JSON payload.
	Variations string `gorm:"type:text;not null"`
	// Variants defines encoded variant JSON payload.
	Variants string `gorm:"type:text;not null"`
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time
	// DeletedAt defines soft-delete timestamps.
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName defines storage table name.
func (productRecord) TableName() string {
	return "products"
}

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

// EnsureSchema migrates product storage schema.
func (r *Repository) EnsureSchema(ctx context.Context) error {
	if err := r.db.WithContext(ctx).AutoMigrate(&productRecord{}); err != nil {
		return fmt.Errorf("migrate product schema: %w", err)
	}

	return nil
}

// Create persists product entities.
func (r *Repository) Create(ctx context.Context, entity *productdomain.Product) error {
	payload, err := toRecordPayload(*entity)
	if err != nil {
		return fmt.Errorf("encode product: %w", err)
	}

	record := productRecord{
		ID:         strings.TrimSpace(entity.ID),
		SKU:        strings.TrimSpace(entity.SKU),
		Gallery:    payload.Gallery,
		Datasheets: payload.Datasheets,
		Variations: payload.Variations,
		Variants:   payload.Variants,
		DeletedAt:  toDeletedAt(entity.DeletedAt),
	}
	if record.ID == "" {
		record.ID = generateID()
	}

	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		if isDuplicateSKUErr(err) {
			return productport.ErrDuplicateSKU
		}
		return fmt.Errorf("create product record: %w", err)
	}

	mapped, mapErr := toDomain(record)
	if mapErr != nil {
		return mapErr
	}
	*entity = mapped
	return nil
}

// GetByID retrieves products by ID.
func (r *Repository) GetByID(ctx context.Context, id string) (*productdomain.Product, error) {
	var record productRecord
	if err := r.db.WithContext(ctx).First(&record, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, productport.ErrNotFound
		}
		return nil, fmt.Errorf("get product record: %w", err)
	}

	mapped, mapErr := toDomain(record)
	if mapErr != nil {
		return nil, mapErr
	}
	return &mapped, nil
}

// List retrieves all non-deleted products.
func (r *Repository) List(ctx context.Context) ([]productdomain.Product, error) {
	records := make([]productRecord, 0)
	if err := r.db.WithContext(ctx).Order("created_at desc").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list product records: %w", err)
	}

	result := make([]productdomain.Product, 0, len(records))
	for _, record := range records {
		mapped, mapErr := toDomain(record)
		if mapErr != nil {
			return nil, mapErr
		}
		result = append(result, mapped)
	}

	return result, nil
}

// Update persists product updates.
func (r *Repository) Update(ctx context.Context, entity *productdomain.Product) error {
	if strings.TrimSpace(entity.ID) == "" {
		return productport.ErrNotFound
	}

	payload, err := toRecordPayload(*entity)
	if err != nil {
		return fmt.Errorf("encode product: %w", err)
	}

	tx := r.db.WithContext(ctx).Model(&productRecord{}).Where("id = ?", entity.ID).Updates(map[string]any{
		"sku":        strings.TrimSpace(entity.SKU),
		"gallery":    payload.Gallery,
		"datasheets": payload.Datasheets,
		"variations": payload.Variations,
		"variants":   payload.Variants,
	})
	if tx.Error != nil {
		if isDuplicateSKUErr(tx.Error) {
			return productport.ErrDuplicateSKU
		}
		return fmt.Errorf("update product record: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return productport.ErrNotFound
	}

	latest, loadErr := r.GetByID(ctx, entity.ID)
	if loadErr != nil {
		return loadErr
	}
	*entity = *latest
	return nil
}

// Delete soft-deletes products by ID.
func (r *Repository) Delete(ctx context.Context, id string) error {
	tx := r.db.WithContext(ctx).Delete(&productRecord{}, "id = ?", id)
	if tx.Error != nil {
		return fmt.Errorf("delete product record: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return productport.ErrNotFound
	}

	return nil
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
