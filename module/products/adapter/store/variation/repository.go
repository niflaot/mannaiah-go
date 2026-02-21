package variation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	variationdomain "mannaiah/module/products/domain/variation"
	variationport "mannaiah/module/products/port/variation"

	"gorm.io/gorm"
)

var (
	// ErrNilDB is returned when DB dependencies are nil.
	ErrNilDB = errors.New("variations db must not be nil")
)

// Repository implements variation persistence using GORM.
type Repository struct {
	// db defines GORM dependencies.
	db *gorm.DB
}

// variationRecord defines variation persistence schema.
type variationRecord struct {
	// ID is the primary key identifier.
	ID string `gorm:"primaryKey;size:64"`
	// Name is the human-readable variation label.
	Name string `gorm:"size:255;not null"`
	// Definition identifies variation type.
	Definition string `gorm:"size:32;not null;index"`
	// Value is the machine-readable variation value.
	Value string `gorm:"size:255;not null"`
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time
	// DeletedAt defines soft-delete timestamps.
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName defines storage table name.
func (variationRecord) TableName() string { return "variations" }

var (
	// _ ensures Repository satisfies variation repository contracts.
	_ variationport.Repository = (*Repository)(nil)
)

// NewRepository creates variation repositories.
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

// Create persists a new variation entity.
func (r *Repository) Create(ctx context.Context, entity *variationdomain.Variation) error {
	record := toRecord(*entity)
	if strings.TrimSpace(record.ID) == "" {
		record.ID = generateID()
	}

	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return fmt.Errorf("create variation record: %w", err)
	}

	*entity = toDomain(record)
	return nil
}

// GetByID retrieves a variation entity by ID.
func (r *Repository) GetByID(ctx context.Context, id string) (*variationdomain.Variation, error) {
	var record variationRecord
	if err := r.db.WithContext(ctx).First(&record, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, variationport.ErrNotFound
		}
		return nil, fmt.Errorf("get variation record: %w", err)
	}

	entity := toDomain(record)
	return &entity, nil
}

// List retrieves all non-deleted variations.
func (r *Repository) List(ctx context.Context) ([]variationdomain.Variation, error) {
	records := make([]variationRecord, 0)
	if err := r.db.WithContext(ctx).Order("created_at desc").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list variation records: %w", err)
	}

	result := make([]variationdomain.Variation, 0, len(records))
	for _, record := range records {
		result = append(result, toDomain(record))
	}

	return result, nil
}

// Update persists modifications for an existing variation.
func (r *Repository) Update(ctx context.Context, entity *variationdomain.Variation) error {
	if strings.TrimSpace(entity.ID) == "" {
		return variationport.ErrNotFound
	}

	record := toRecord(*entity)
	tx := r.db.WithContext(ctx).Model(&variationRecord{}).Where("id = ?", record.ID).Updates(map[string]any{
		"name":  record.Name,
		"value": record.Value,
	})
	if tx.Error != nil {
		return fmt.Errorf("update variation record: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return variationport.ErrNotFound
	}

	latest, err := r.GetByID(ctx, record.ID)
	if err != nil {
		return err
	}
	*entity = *latest
	return nil
}

// Delete soft-deletes a variation by ID.
func (r *Repository) Delete(ctx context.Context, id string) error {
	tx := r.db.WithContext(ctx).Delete(&variationRecord{}, "id = ?", id)
	if tx.Error != nil {
		return fmt.Errorf("delete variation record: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return variationport.ErrNotFound
	}

	return nil
}

// toRecord maps domain variation entities to persistence records.
func toRecord(entity variationdomain.Variation) variationRecord {
	return variationRecord{
		ID:         strings.TrimSpace(entity.ID),
		Name:       strings.TrimSpace(entity.Name),
		Definition: strings.TrimSpace(string(entity.Definition)),
		Value:      strings.TrimSpace(entity.Value),
		CreatedAt:  entity.CreatedAt,
		UpdatedAt:  entity.UpdatedAt,
		DeletedAt:  toDeletedAt(entity.DeletedAt),
	}
}

// toDomain maps persistence records to domain variation entities.
func toDomain(record variationRecord) variationdomain.Variation {
	return variationdomain.Variation{
		ID:         record.ID,
		Name:       record.Name,
		Definition: variationdomain.Definition(record.Definition),
		Value:      record.Value,
		CreatedAt:  record.CreatedAt,
		UpdatedAt:  record.UpdatedAt,
		IsDeleted:  record.DeletedAt.Valid,
		DeletedAt:  fromDeletedAt(record.DeletedAt),
	}
}

// generateID creates random variation identifiers.
func generateID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}

	return hex.EncodeToString(bytes)
}

// toDeletedAt converts deleted-at pointers into GORM deleted-at values.
func toDeletedAt(value *time.Time) gorm.DeletedAt {
	if value == nil {
		return gorm.DeletedAt{}
	}

	return gorm.DeletedAt{Time: *value, Valid: true}
}

// fromDeletedAt converts GORM deleted-at values into nullable timestamps.
func fromDeletedAt(value gorm.DeletedAt) *time.Time {
	if !value.Valid {
		return nil
	}

	timestamp := value.Time
	return &timestamp
}
