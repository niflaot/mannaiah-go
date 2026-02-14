package product

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"
	productdomain "mannaiah/module/products/domain/product"
)

type legacyProductRow struct {
	ID         string
	Gallery    string
	Datasheets string
	Variations string
	Variants   string
}

// migrateLegacyRelations migrates legacy JSON columns into normalized product relation tables.
func (r *Repository) migrateLegacyRelations(ctx context.Context) error {
	migrator := r.db.WithContext(ctx).Migrator()
	if !migrator.HasColumn("products", "gallery") &&
		!migrator.HasColumn("products", "datasheets") &&
		!migrator.HasColumn("products", "variations") &&
		!migrator.HasColumn("products", "variants") {
		return nil
	}

	rows := make([]legacyProductRow, 0)
	if err := r.db.WithContext(ctx).Table("products").Select("id, gallery, datasheets, variations, variants").Find(&rows).Error; err != nil {
		return fmt.Errorf("load legacy product rows: %w", err)
	}
	for _, row := range rows {
		needsMigration, err := r.productNeedsLegacyMigration(ctx, row.ID)
		if err != nil {
			return err
		}
		if !needsMigration {
			continue
		}

		entity, err := parseLegacyProductRow(row)
		if err != nil {
			return fmt.Errorf("parse legacy product row %q: %w", row.ID, err)
		}
		if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return replaceProductRelations(tx, row.ID, entity)
		}); err != nil {
			return fmt.Errorf("migrate legacy product relations for %q: %w", row.ID, err)
		}
	}

	return nil
}

// productNeedsLegacyMigration reports whether target products have normalized relation rows.
func (r *Repository) productNeedsLegacyMigration(ctx context.Context, productID string) (bool, error) {
	trimmedID := strings.TrimSpace(productID)

	counts := []struct {
		model any
		query string
	}{
		{model: &productGalleryRecord{}, query: "product_id = ?"},
		{model: &productDatasheetRecord{}, query: "product_id = ?"},
		{model: &productVariationLinkRecord{}, query: "product_id = ?"},
		{model: &productVariantRecord{}, query: "product_id = ?"},
	}
	for _, value := range counts {
		var count int64
		if err := r.db.WithContext(ctx).Model(value.model).Where(value.query, trimmedID).Count(&count).Error; err != nil {
			return false, fmt.Errorf("count normalized product relations: %w", err)
		}
		if count > 0 {
			return false, nil
		}
	}

	return true, nil
}

// parseLegacyProductRow parses legacy JSON fields into aggregate relation values.
func parseLegacyProductRow(row legacyProductRow) (productdomain.Product, error) {
	entity := productdomain.Product{}
	if err := unmarshalLegacyJSON(row.Gallery, &entity.Gallery); err != nil {
		return productdomain.Product{}, fmt.Errorf("unmarshal gallery: %w", err)
	}
	if err := unmarshalLegacyJSON(row.Datasheets, &entity.Datasheets); err != nil {
		return productdomain.Product{}, fmt.Errorf("unmarshal datasheets: %w", err)
	}
	if err := unmarshalLegacyJSON(row.Variations, &entity.Variations); err != nil {
		return productdomain.Product{}, fmt.Errorf("unmarshal variations: %w", err)
	}
	if err := unmarshalLegacyJSON(row.Variants, &entity.Variants); err != nil {
		return productdomain.Product{}, fmt.Errorf("unmarshal variants: %w", err)
	}
	entity.Normalize()

	return entity, nil
}

// unmarshalLegacyJSON unmarshals legacy JSON strings into destination values.
func unmarshalLegacyJSON(raw string, destination any) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		trimmed = "[]"
	}

	return json.Unmarshal([]byte(trimmed), destination)
}
