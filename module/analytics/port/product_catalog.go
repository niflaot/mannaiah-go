package port

import "context"

// ProductCatalogEntry defines a minimal product record for recommendation filtering.
type ProductCatalogEntry struct {
	// ID is the product identifier.
	ID string
	// Price is the product price (zero if unset).
	Price float64
	// Tags contains all active product tag values.
	Tags []string
	// Datasheets contains realm-scoped display data.
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
}

// ProductGalleryEntry defines one gallery item for a product.
type ProductGalleryEntry struct {
	// AssetID is the referenced asset identifier.
	AssetID string
	// IncludedRealms defines realms where this image is visible (empty means all realms).
	IncludedRealms []string
	// IsMain reports whether this is the primary product image.
	IsMain bool
}

// ProductCatalogStore defines read behavior over the product catalog for recommendation resolution.
type ProductCatalogStore interface {
	// GetProductsByBaseTag returns active products that have baseTag.
	// When expandedTags is non-empty, only products that also have at least one expanded tag are returned.
	// When categoryID is non-empty, results are restricted to that category.
	// When excludeIDs is non-empty, those product IDs are excluded from results.
	// Limit constrains the maximum number of returned entries.
	GetProductsByBaseTag(ctx context.Context, baseTag string, expandedTags []string, categoryID string, excludeIDs []string, limit int) ([]ProductCatalogEntry, error)
	// GetProductsByIDs returns active products for the given product IDs, preserving input order.
	// Used to load pinned products. Returns only IDs that exist and are not soft-deleted.
	GetProductsByIDs(ctx context.Context, ids []string) ([]ProductCatalogEntry, error)
}
