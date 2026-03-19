package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"mannaiah/module/analytics/port"
)

// productTagSeedRow holds a product-tag row read during taxonomy seed.
type productTagSeedRow struct {
	// ProductID is the product UUID.
	ProductID string `gorm:"column:product_id"`
	// Tag is the product tag value.
	Tag string `gorm:"column:tag"`
}

// categoryProductSeedRow holds a category-product mapping row read during taxonomy seed.
type categoryProductSeedRow struct {
	// ProductID is the product UUID.
	ProductID string `gorm:"column:product_id"`
	// CategoryID is the category UUID.
	CategoryID string `gorm:"column:category_id"`
	// CategoryName is the human-readable category name.
	CategoryName string `gorm:"column:category_name"`
}

// variationTaxonomySeedRow holds a product variation row read during taxonomy seed.
type variationTaxonomySeedRow struct {
	// ProductID is the product UUID.
	ProductID string `gorm:"column:product_id"`
	// SKU is the variant SKU.
	SKU string `gorm:"column:sku"`
	// VariationID is the variation identifier.
	VariationID string `gorm:"column:variation_id"`
	// VariationName is the variation attribute name.
	VariationName string `gorm:"column:variation_name"`
	// VariationValue is the variation attribute value.
	VariationValue string `gorm:"column:variation_value"`
}

// seedProductTaxonomy reads products, product_tags, and category_products from the transactional
// database and batch-upserts product taxonomy rows into ClickHouse.
func (s *AnalyticsService) seedProductTaxonomy(ctx context.Context, taxonomyStore port.TaxonomyStore) error {
	now := time.Now().UTC()

	tagRows := make([]productTagSeedRow, 0, seedBatchSize)
	if err := s.db.WithContext(ctx).
		Table("product_tags").
		Select("product_tags.product_id", "tags.name AS tag").
		Joins("JOIN tags ON tags.id = product_tags.tag_id AND tags.deleted_at IS NULL").
		Order("product_tags.product_id ASC").
		Scan(&tagRows).Error; err != nil {
		return fmt.Errorf("seed product tags: %w", err)
	}

	catRows := make([]categoryProductSeedRow, 0, seedBatchSize)
	query := `SELECT cp.product_id, cp.category_id, c.name AS category_name
		FROM category_products cp
		INNER JOIN categories c ON c.id = cp.category_id
		WHERE c.deleted_at IS NULL`
	if err := s.db.WithContext(ctx).Raw(query).Scan(&catRows).Error; err != nil {
		return fmt.Errorf("seed category products: %w", err)
	}

	payload := make([]port.ProductTaxonomyRow, 0, len(tagRows)+len(catRows))
	for _, row := range tagRows {
		pid := strings.TrimSpace(row.ProductID)
		tag := strings.TrimSpace(row.Tag)
		if pid == "" || tag == "" {
			continue
		}
		payload = append(payload, port.ProductTaxonomyRow{
			ProductID: pid,
			Tag:       tag,
			UpdatedAt: now,
		})
	}
	for _, row := range catRows {
		pid := strings.TrimSpace(row.ProductID)
		cid := strings.TrimSpace(row.CategoryID)
		if pid == "" || cid == "" {
			continue
		}
		payload = append(payload, port.ProductTaxonomyRow{
			ProductID:    pid,
			CategoryID:   cid,
			CategoryName: strings.TrimSpace(row.CategoryName),
			UpdatedAt:    now,
		})
	}

	for i := 0; i < len(payload); i += seedBatchSize {
		end := i + seedBatchSize
		if end > len(payload) {
			end = len(payload)
		}
		if err := taxonomyStore.UpsertProductTaxonomy(ctx, payload[i:end]); err != nil {
			return fmt.Errorf("upsert product taxonomy batch: %w", err)
		}
	}

	return nil
}

// seedVariationTaxonomy reads product_variants, product_variant_variations, and variations from
// the transactional database and batch-upserts variation taxonomy rows into ClickHouse.
func (s *AnalyticsService) seedVariationTaxonomy(ctx context.Context, taxonomyStore port.TaxonomyStore) error {
	now := time.Now().UTC()

	varRows := make([]variationTaxonomySeedRow, 0, seedBatchSize)
	query := `SELECT pv.product_id, pv.sku, pvv.variation_id, v.name AS variation_name, v.value AS variation_value
		FROM product_variants pv
		JOIN product_variant_variations pvv ON pvv.variant_id = pv.id
		JOIN variations v ON v.id = pvv.variation_id
		WHERE pv.sku != '' AND v.name != '' AND v.value != ''`
	if err := s.db.WithContext(ctx).Raw(query).Scan(&varRows).Error; err != nil {
		return fmt.Errorf("seed variation taxonomy: %w", err)
	}

	payload := make([]port.VariationTaxonomyRow, 0, len(varRows))
	for _, row := range varRows {
		pid := strings.TrimSpace(row.ProductID)
		vid := strings.TrimSpace(row.VariationID)
		if pid == "" || vid == "" {
			continue
		}
		payload = append(payload, port.VariationTaxonomyRow{
			ProductID:      pid,
			SKU:            strings.TrimSpace(row.SKU),
			VariationID:    vid,
			VariationName:  strings.TrimSpace(row.VariationName),
			VariationValue: strings.TrimSpace(row.VariationValue),
			UpdatedAt:      now,
		})
	}

	for i := 0; i < len(payload); i += seedBatchSize {
		end := i + seedBatchSize
		if end > len(payload) {
			end = len(payload)
		}
		if err := taxonomyStore.UpsertVariationTaxonomy(ctx, payload[i:end]); err != nil {
			return fmt.Errorf("upsert variation taxonomy batch: %w", err)
		}
	}

	return nil
}
