package product

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
	productdomain "mannaiah/module/products/domain/product"
)

// replaceProductRelations replaces all product relation rows from aggregate state.
func replaceProductRelations(tx *gorm.DB, productID string, entity productdomain.Product) error {
	if err := clearProductRelations(tx, productID); err != nil {
		return err
	}

	if err := createGalleryRelations(tx, productID, entity.Gallery); err != nil {
		return err
	}
	if err := createDatasheetRelations(tx, productID, entity.Datasheets); err != nil {
		return err
	}
	if err := createVariationLinks(tx, productID, entity.Variations); err != nil {
		return err
	}
	if err := createVariantRelations(tx, productID, entity.Variants); err != nil {
		return err
	}
	if err := replaceProductTags(tx, productID, entity.Tags); err != nil {
		return err
	}

	return nil
}

// clearProductRelations deletes all child relation rows for a product.
func clearProductRelations(tx *gorm.DB, productID string) error {
	trimmedID := strings.TrimSpace(productID)

	galleryIDs := make([]uint, 0)
	if err := tx.Model(&productGalleryRecord{}).Where("product_id = ?", trimmedID).Pluck("id", &galleryIDs).Error; err != nil {
		return fmt.Errorf("list gallery relation ids: %w", err)
	}
	if len(galleryIDs) > 0 {
		if err := tx.Where("gallery_item_id IN ?", galleryIDs).Delete(&productGalleryIncludedRealmRecord{}).Error; err != nil {
			return fmt.Errorf("delete gallery included realm relations: %w", err)
		}
		if err := tx.Where("gallery_item_id IN ?", galleryIDs).Delete(&productGalleryVariationRecord{}).Error; err != nil {
			return fmt.Errorf("delete gallery variation relations: %w", err)
		}
	}
	if err := tx.Where("product_id = ?", trimmedID).Delete(&productGalleryRecord{}).Error; err != nil {
		return fmt.Errorf("delete gallery relations: %w", err)
	}

	datasheetIDs := make([]uint, 0)
	if err := tx.Model(&productDatasheetRecord{}).Where("product_id = ?", trimmedID).Pluck("id", &datasheetIDs).Error; err != nil {
		return fmt.Errorf("list datasheet relation ids: %w", err)
	}
	if len(datasheetIDs) > 0 {
		if err := tx.Where("datasheet_id IN ?", datasheetIDs).Delete(&productDatasheetAttributeRecord{}).Error; err != nil {
			return fmt.Errorf("delete datasheet attribute relations: %w", err)
		}
	}
	if err := tx.Where("product_id = ?", trimmedID).Delete(&productDatasheetRecord{}).Error; err != nil {
		return fmt.Errorf("delete datasheet relations: %w", err)
	}

	variantIDs := make([]uint, 0)
	if err := tx.Model(&productVariantRecord{}).Where("product_id = ?", trimmedID).Pluck("id", &variantIDs).Error; err != nil {
		return fmt.Errorf("list variant relation ids: %w", err)
	}
	if len(variantIDs) > 0 {
		if err := tx.Where("variant_id IN ?", variantIDs).Delete(&productVariantVariationRecord{}).Error; err != nil {
			return fmt.Errorf("delete variant variation relations: %w", err)
		}
	}
	if err := tx.Where("product_id = ?", trimmedID).Delete(&productVariantRecord{}).Error; err != nil {
		return fmt.Errorf("delete variant relations: %w", err)
	}

	if err := tx.Where("product_id = ?", trimmedID).Delete(&productVariationLinkRecord{}).Error; err != nil {
		return fmt.Errorf("delete product variation links: %w", err)
	}

	return nil
}

// createGalleryRelations persists gallery relation rows.
func createGalleryRelations(tx *gorm.DB, productID string, values []productdomain.GalleryItem) error {
	for index, item := range values {
		position := resolveGalleryPosition(item.Position, index)
		variationPosition := resolveGalleryVariationPosition(item.VariationPosition, len(item.VariationIDs) > 0)
		galleryRecord := productGalleryRecord{
			ProductID:         strings.TrimSpace(productID),
			Position:          position,
			VariationPosition: variationPosition,
			AssetID:           strings.TrimSpace(item.AssetID),
			IsMain:            item.IsMain,
		}
		if err := tx.Create(&galleryRecord).Error; err != nil {
			return fmt.Errorf("create gallery relation: %w", err)
		}

		for includedIndex, includedRealm := range item.IncludedRealms {
			includedRecord := productGalleryIncludedRealmRecord{
				GalleryItemID: galleryRecord.ID,
				Position:      includedIndex,
				Realm:         strings.TrimSpace(includedRealm),
			}
			if err := tx.Create(&includedRecord).Error; err != nil {
				return fmt.Errorf("create gallery included realm relation: %w", err)
			}
		}
		for variationIndex, variationID := range item.VariationIDs {
			variationRecord := productGalleryVariationRecord{
				GalleryItemID: galleryRecord.ID,
				Position:      variationIndex,
				VariationID:   strings.TrimSpace(variationID),
			}
			if err := tx.Create(&variationRecord).Error; err != nil {
				return fmt.Errorf("create gallery variation relation: %w", err)
			}
		}
	}

	return nil
}

// resolveGalleryPosition resolves persisted gallery position values from explicit payload values or fallback index values.
func resolveGalleryPosition(value *int, fallback int) int {
	if value == nil {
		return fallback
	}

	resolved := *value
	if resolved < 0 {
		return 0
	}

	return resolved
}

// resolveGalleryVariationPosition resolves persisted variation-scoped gallery positions for variation-linked images.
func resolveGalleryVariationPosition(value *int, hasVariationIDs bool) *int {
	if !hasVariationIDs || value == nil {
		return nil
	}

	resolved := *value
	if resolved < 0 {
		resolved = 0
	}

	return &resolved
}

// createDatasheetRelations persists datasheet relation rows.
func createDatasheetRelations(tx *gorm.DB, productID string, values []productdomain.Datasheet) error {
	for index, item := range values {
		datasheetRecord := productDatasheetRecord{
			ProductID:   strings.TrimSpace(productID),
			Position:    index,
			Realm:       strings.TrimSpace(item.Realm),
			Name:        strings.TrimSpace(item.Name),
			Description: strings.TrimSpace(item.Description),
		}
		if err := tx.Create(&datasheetRecord).Error; err != nil {
			return fmt.Errorf("create datasheet relation: %w", err)
		}

		for key, value := range item.Attributes {
			encodedValue, err := marshalAttributeValue(value)
			if err != nil {
				return err
			}
			attributeRecord := productDatasheetAttributeRecord{
				DatasheetID: datasheetRecord.ID,
				Key:         strings.TrimSpace(key),
				ValueJSON:   encodedValue,
			}
			if err := tx.Create(&attributeRecord).Error; err != nil {
				return fmt.Errorf("create datasheet attribute relation: %w", err)
			}
		}
	}

	return nil
}

// createVariationLinks persists product-level variation links.
func createVariationLinks(tx *gorm.DB, productID string, values []string) error {
	for index, variationID := range values {
		record := productVariationLinkRecord{
			ProductID:   strings.TrimSpace(productID),
			Position:    index,
			VariationID: strings.TrimSpace(variationID),
		}
		if err := tx.Create(&record).Error; err != nil {
			return fmt.Errorf("create product variation link: %w", err)
		}
	}

	return nil
}

// createVariantRelations persists variant relation rows.
func createVariantRelations(tx *gorm.DB, productID string, values []productdomain.Variant) error {
	for index, item := range values {
		variantRecord := productVariantRecord{
			ProductID: strings.TrimSpace(productID),
			Position:  index,
			SKU:       strings.TrimSpace(item.SKU),
		}
		if err := tx.Create(&variantRecord).Error; err != nil {
			return fmt.Errorf("create variant relation: %w", err)
		}

		for variationIndex, variationID := range item.VariationIDs {
			variantVariationRecord := productVariantVariationRecord{
				VariantID:   variantRecord.ID,
				Position:    variationIndex,
				VariationID: strings.TrimSpace(variationID),
			}
			if err := tx.Create(&variantVariationRecord).Error; err != nil {
				return fmt.Errorf("create variant variation relation: %w", err)
			}
		}
	}

	return nil
}
