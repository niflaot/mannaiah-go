package product

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// productTagRecord defines product tag rows for taxonomy classification.
type productTagRecord struct {
	// ID defines surrogate identifiers.
	ID uint `gorm:"primaryKey"`
	// ProductID defines owning product identifiers.
	ProductID string `gorm:"size:64;not null;index"`
	// Position defines stable tag ordering.
	Position int `gorm:"not null;index"`
	// TagID defines the foreign key reference to the canonical tags registry.
	TagID uint `gorm:"not null;index"`
}

// productTagLoadRow defines the result of joining product_tags with tags for read operations.
type productTagLoadRow struct {
	// Position defines stable tag ordering.
	Position int
	// Name defines the canonical tag name resolved from the tags registry.
	Name string
}

// TableName defines storage table name.
func (productTagRecord) TableName() string { return "product_tags" }

// replaceProductTags replaces all tag rows for a product from aggregate state.
// Each tag name is resolved to its canonical ID from the tags registry before insertion.
func replaceProductTags(tx *gorm.DB, productID string, tags []string) error {
	trimmedID := strings.TrimSpace(productID)
	if err := tx.Where("product_id = ?", trimmedID).Delete(&productTagRecord{}).Error; err != nil {
		return fmt.Errorf("delete product tag relations: %w", err)
	}

	for index, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed == "" {
			continue
		}

		var tagID uint
		if err := tx.Raw("SELECT id FROM tags WHERE name = ? AND deleted_at IS NULL LIMIT 1", trimmed).Scan(&tagID).Error; err != nil {
			return fmt.Errorf("resolve tag id for %q: %w", trimmed, err)
		}
		if tagID == 0 {
			return fmt.Errorf("tag %q not found in canonical registry", trimmed)
		}

		record := productTagRecord{
			ProductID: trimmedID,
			Position:  index,
			TagID:     tagID,
		}
		if err := tx.Create(&record).Error; err != nil {
			return fmt.Errorf("create product tag relation: %w", err)
		}
	}

	return nil
}
