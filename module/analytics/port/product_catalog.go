package port

import "context"

// ProductCatalogEntry defines a minimal product record for recommendation filtering.
type ProductCatalogEntry struct {
	// ID is the product identifier.
	ID string
	// Price is the base product price (fallback; prefer realm datasheet price for display).
	Price float64
	// Tags contains all active product tag values.
	Tags []string
	// VariationIDs contains the variation IDs linked to this product via product_variation_links.
	VariationIDs []string
	// Datasheets contains realm-scoped display data ordered by position.
	Datasheets []ProductDatasheetEntry
	// Gallery contains gallery image entries ordered by position.
	Gallery []ProductGalleryEntry
}

// ProductDatasheetEntry defines realm-scoped display data for a product.
type ProductDatasheetEntry struct {
	// Realm identifies the display context.
	Realm string
	// Name is the realm-specific product display name.
	Name string
	// Price is the realm-specific price parsed from product_datasheet_attributes key="price".
	// Nil when the attribute is absent or cannot be parsed as a number.
	Price *float64
}

// ProductGalleryEntry defines one gallery item for a product.
type ProductGalleryEntry struct {
	// AssetID is the referenced asset identifier.
	AssetID string
	// IncludedRealms defines realms where this image is visible (empty means all realms).
	IncludedRealms []string
	// IsMain reports whether this is the primary product image.
	IsMain bool
	// VariationIDs lists variation IDs this gallery item is linked to via product_gallery_variations.
	// Empty means the image is not variation-specific and is shown for all variations.
	VariationIDs []string
}

// ProductCatalogStore defines read behavior over the product catalog for recommendation resolution.
type ProductCatalogStore interface {
	// GetProductsByBaseTag returns active products that have baseTag.
	// When expandedTags is non-empty, only products sharing at least one expanded tag are returned.
	// When categoryID is non-empty, results are restricted to that category.
	// When excludeIDs is non-empty, those product IDs are excluded.
	// When filterVariationIDs is non-empty, only products with at least one matching variation are returned.
	// Limit constrains the maximum number of returned entries.
	GetProductsByBaseTag(ctx context.Context, baseTag string, expandedTags []string, categoryID string, excludeIDs []string, filterVariationIDs []string, limit int) ([]ProductCatalogEntry, error)
	// GetProductsByIDs returns active products for the given product IDs, preserving input order.
	// When filterVariationIDs is non-empty, only products with at least one matching variation are returned.
	GetProductsByIDs(ctx context.Context, ids []string, filterVariationIDs []string) ([]ProductCatalogEntry, error)
}
