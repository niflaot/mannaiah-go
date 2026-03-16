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
	// Tag defines the tag value.
	Tag string `gorm:"size:128;not null"`
}

// TableName defines storage table name.
func (productTagRecord) TableName() string { return "product_tags" }

// replaceProductTags replaces all tag rows for a product from aggregate state.
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
		record := productTagRecord{
			ProductID: trimmedID,
			Position:  index,
			Tag:       trimmed,
		}
		if err := tx.Create(&record).Error; err != nil {
			return fmt.Errorf("create product tag relation: %w", err)
		}
	}

	return nil
}
