package port

import "context"

// SyncProductRequest defines Falabella product-sync request values.
type SyncProductRequest struct {
	// SKU defines seller SKU values used as Falabella product identifiers.
	SKU string
	// ParentSKU defines parent seller SKU values for variant products.
	ParentSKU string
	// Variation defines Falabella variation relationship values.
	Variation string
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
	// OperatorCode defines Falabella business-unit operator code values.
	OperatorCode string
	// Attributes defines additional ProductData field values.
	Attributes map[string]string
}

// SyncProductImagesRequest defines Falabella product-image sync request values.
type SyncProductImagesRequest struct {
	// SKU defines seller SKU values used as Falabella product identifiers.
	SKU string
	// URLs defines image URL values to associate with the provided SKU.
	URLs []string
}

// Source defines Falabella integration source behavior.
type Source interface {
	// Validate verifies integration availability.
	Validate(ctx context.Context) error
	// GetBrands retrieves raw JSON payload returned by Falabella GetBrands.
	GetBrands(ctx context.Context) ([]byte, error)
	// SyncProduct upserts a product into Falabella from mapped marketplace values.
	SyncProduct(ctx context.Context, request SyncProductRequest) ([]byte, error)
	// SyncProductImages configures product images in Falabella for one SKU.
	SyncProductImages(ctx context.Context, request SyncProductImagesRequest) ([]byte, error)
	// GetFeedStatus retrieves Falabella feed status by feed identifier.
	GetFeedStatus(ctx context.Context, feedID string) ([]byte, error)
}
