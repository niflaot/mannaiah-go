package port

import "context"

// SyncProductRequest defines Falabella product-sync request values.
type SyncProductRequest struct {
	// SKU defines seller SKU values used as Falabella product identifiers.
	SKU string
	// Name defines product display names.
	Name string
	// Brand defines Falabella brand values.
	Brand string
	// Model defines product model values.
	Model string
	// Description defines product description values.
	Description string
	// PrimaryCategory defines Falabella category identifier values.
	PrimaryCategory string
	// TaxClass defines tax-class percentage values.
	TaxClass string
	// Price defines marketplace product price values.
	Price string
	// SalePrice defines optional sale-price values.
	SalePrice string
	// SaleStartDate defines optional sale start-date values.
	SaleStartDate string
	// SaleEndDate defines optional sale end-date values.
	SaleEndDate string
	// Attributes defines additional ProductData field values.
	Attributes map[string]string
}

// Source defines Falabella integration source behavior.
type Source interface {
	// Validate verifies integration availability.
	Validate(ctx context.Context) error
	// GetBrands retrieves raw JSON payload returned by Falabella GetBrands.
	GetBrands(ctx context.Context) ([]byte, error)
	// SyncProduct upserts a product into Falabella from mapped marketplace values.
	SyncProduct(ctx context.Context, request SyncProductRequest) ([]byte, error)
}
