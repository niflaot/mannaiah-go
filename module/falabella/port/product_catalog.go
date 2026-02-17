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

// CatalogProduct defines product values consumed by Falabella sync use cases.
type CatalogProduct struct {
	// ID defines product identifiers.
	ID string
	// SKU defines seller SKU identifiers.
	SKU string
	// Datasheets defines realm-scoped datasheet values.
	Datasheets []CatalogDatasheet
}

// ProductCatalog defines cross-module product lookup behavior used by Falabella sync services.
type ProductCatalog interface {
	// GetProduct retrieves products by identifier.
	GetProduct(ctx context.Context, id string) (*CatalogProduct, error)
	// ListProducts lists all products.
	ListProducts(ctx context.Context) ([]CatalogProduct, error)
}

