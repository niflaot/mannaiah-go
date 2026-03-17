package port

import (
	"context"
	"time"
)

// ProductTaxonomyRow defines a product taxonomy row for batch upsert.
type ProductTaxonomyRow struct {
	// ProductID identifies the product.
	ProductID string
	// Tag is an optional product tag value.
	Tag string
	// CategoryID is an optional product category identifier.
	CategoryID string
	// CategoryName is the optional human-readable category name.
	CategoryName string
	// UpdatedAt is the row freshness timestamp.
	UpdatedAt time.Time
}

// VariationTaxonomyRow defines a product variation taxonomy row for batch upsert.
type VariationTaxonomyRow struct {
	// ProductID identifies the parent product.
	ProductID string
	// SKU is the variant SKU.
	SKU string
	// VariationID identifies the variation attribute.
	VariationID string
	// VariationName is the variation attribute name (e.g. "color").
	VariationName string
	// VariationValue is the variation attribute value (e.g. "black").
	VariationValue string
	// UpdatedAt is the row freshness timestamp.
	UpdatedAt time.Time
}

// TaxonomyStore defines ClickHouse-backed product taxonomy persistence behavior.
type TaxonomyStore interface {
	// UpsertProductTaxonomy batch-upserts product taxonomy rows.
	UpsertProductTaxonomy(ctx context.Context, rows []ProductTaxonomyRow) error
	// UpsertVariationTaxonomy batch-upserts product variation taxonomy rows.
	UpsertVariationTaxonomy(ctx context.Context, rows []VariationTaxonomyRow) error
}
