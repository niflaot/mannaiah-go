package product

import (
	"context"
	"fmt"

	productdomain "mannaiah/module/products/domain/product"
)

// loadProductAggregate loads a product aggregate root and its normalized relations.
func (r *Repository) loadProductAggregate(ctx context.Context, record productRecord) (productdomain.Product, error) {
	entity := productdomain.Product{
		ID:        record.ID,
		SKU:       record.SKU,
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
		IsDeleted: record.DeletedAt.Valid,
		DeletedAt: fromDeletedAt(record.DeletedAt),
	}

	galleryRows := make([]productGalleryRecord, 0)
	if err := r.db.WithContext(ctx).Where("product_id = ?", record.ID).Order("position ASC").Find(&galleryRows).Error; err != nil {
		return productdomain.Product{}, fmt.Errorf("load product gallery relations: %w", err)
	}
	for _, galleryRow := range galleryRows {
		item := productdomain.GalleryItem{
			AssetID: galleryRow.AssetID,
			IsMain:  galleryRow.IsMain,
		}

		excludedRows := make([]productGalleryExcludedRealmRecord, 0)
		if err := r.db.WithContext(ctx).Where("gallery_item_id = ?", galleryRow.ID).Order("position ASC").Find(&excludedRows).Error; err != nil {
			return productdomain.Product{}, fmt.Errorf("load gallery excluded realm relations: %w", err)
		}
		for _, excludedRow := range excludedRows {
			item.ExcludedRealms = append(item.ExcludedRealms, excludedRow.Realm)
		}

		variationRows := make([]productGalleryVariationRecord, 0)
		if err := r.db.WithContext(ctx).Where("gallery_item_id = ?", galleryRow.ID).Order("position ASC").Find(&variationRows).Error; err != nil {
			return productdomain.Product{}, fmt.Errorf("load gallery variation relations: %w", err)
		}
		for _, variationRow := range variationRows {
			item.VariationIDs = append(item.VariationIDs, variationRow.VariationID)
		}

		entity.Gallery = append(entity.Gallery, item)
	}

	datasheetRows := make([]productDatasheetRecord, 0)
	if err := r.db.WithContext(ctx).Where("product_id = ?", record.ID).Order("position ASC").Find(&datasheetRows).Error; err != nil {
		return productdomain.Product{}, fmt.Errorf("load product datasheet relations: %w", err)
	}
	for _, datasheetRow := range datasheetRows {
		item := productdomain.Datasheet{
			Realm:       datasheetRow.Realm,
			Name:        datasheetRow.Name,
			Description: datasheetRow.Description,
			Attributes:  map[string]any{},
		}

		attributeRows := make([]productDatasheetAttributeRecord, 0)
		if err := r.db.WithContext(ctx).Where("datasheet_id = ?", datasheetRow.ID).Order("id ASC").Find(&attributeRows).Error; err != nil {
			return productdomain.Product{}, fmt.Errorf("load datasheet attribute relations: %w", err)
		}
		for _, attributeRow := range attributeRows {
			value, err := unmarshalAttributeValue(attributeRow.ValueJSON)
			if err != nil {
				return productdomain.Product{}, err
			}
			item.Attributes[attributeRow.Key] = value
		}

		entity.Datasheets = append(entity.Datasheets, item)
	}

	variationLinkRows := make([]productVariationLinkRecord, 0)
	if err := r.db.WithContext(ctx).Where("product_id = ?", record.ID).Order("position ASC").Find(&variationLinkRows).Error; err != nil {
		return productdomain.Product{}, fmt.Errorf("load product variation link relations: %w", err)
	}
	for _, variationLinkRow := range variationLinkRows {
		entity.Variations = append(entity.Variations, variationLinkRow.VariationID)
	}

	variantRows := make([]productVariantRecord, 0)
	if err := r.db.WithContext(ctx).Where("product_id = ?", record.ID).Order("position ASC").Find(&variantRows).Error; err != nil {
		return productdomain.Product{}, fmt.Errorf("load product variant relations: %w", err)
	}
	for _, variantRow := range variantRows {
		item := productdomain.Variant{
			SKU: variantRow.SKU,
		}
		variantVariationRows := make([]productVariantVariationRecord, 0)
		if err := r.db.WithContext(ctx).Where("variant_id = ?", variantRow.ID).Order("position ASC").Find(&variantVariationRows).Error; err != nil {
			return productdomain.Product{}, fmt.Errorf("load product variant variation relations: %w", err)
		}
		for _, variantVariationRow := range variantVariationRows {
			item.VariationIDs = append(item.VariationIDs, variantVariationRow.VariationID)
		}
		entity.Variants = append(entity.Variants, item)
	}

	entity.Normalize()

	return entity, nil
}
