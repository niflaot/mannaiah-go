package port

import "context"

// CatalogDatasheet defines product datasheet values consumed by Falabella sync use cases.
type CatalogDatasheet struct {
	// Realm defines datasheet realm identifiers.
	Realm string
	// Name defines realm-scoped product names.
	Name string
	// Description defines realm-scoped product descriptions.
	Description string
	// Attributes defines realm-scoped key-value attributes.
	Attributes map[string]any
}

// CatalogVariation defines resolved variation values for a product variant.
type CatalogVariation struct {
	// ID defines variation identifiers.
	ID string
	// Name defines variation labels.
	Name string
	// Definition defines variation type values.
	Definition string
	// Value defines machine-readable variation values.
	Value string
}

// CatalogVariant defines variant values consumed by Falabella sync use cases.
type CatalogVariant struct {
	// SKU defines variant SKU values.
	SKU string
	// VariationIDs defines linked variation identifier values.
	VariationIDs []string
	// Variations defines resolved variation values.
	Variations []CatalogVariation
}

// CatalogImage defines product-gallery image values for Falabella sync use cases.
type CatalogImage struct {
	// URL defines public image URL values.
	URL string
	// ExcludedRealms defines realm identifiers where this image must be excluded.
	ExcludedRealms []string
	// VariationIDs defines optional linked variation identifier values.
	VariationIDs []string
}

// CatalogProduct defines product values consumed by Falabella sync use cases.
type CatalogProduct struct {
	// ID defines product identifiers.
	ID string
	// SKU defines seller SKU identifiers.
	SKU string
	// Datasheets defines realm-scoped datasheet values.
	Datasheets []CatalogDatasheet
	// Variants defines product variant values.
	Variants []CatalogVariant
	// Images defines product gallery image values.
	Images []CatalogImage
}

// ProductCatalog defines cross-module product lookup behavior used by Falabella sync services.
type ProductCatalog interface {
	// GetProduct retrieves products by identifier.
	GetProduct(ctx context.Context, id string) (*CatalogProduct, error)
	// ListProducts lists all products.
	ListProducts(ctx context.Context) ([]CatalogProduct, error)
}
