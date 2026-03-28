package port

import "context"

// OrderQuotationSource resolves order data required for order-based quotation workflows.
type OrderQuotationSource interface {
	// GetByIDOrIdentifier resolves order quotation data by internal ID or external identifier.
	GetByIDOrIdentifier(ctx context.Context, identifier string) (*OrderQuotationData, error)
}

// OrderQuotationData defines the order-level data needed to build a quotation from an order.
type OrderQuotationData struct {
	// OrderID defines the internal order identifier.
	OrderID string
	// OrderIdentifier defines the external order identifier (e.g. WooCommerce order number).
	OrderIdentifier string
	// DestCityCode defines the destination city code from the order shipping address.
	DestCityCode string
	// TotalValue defines the monetary total of all order items (used as COD amount).
	TotalValue float64
	// Items defines the line items belonging to the order.
	Items []OrderQuotationItem
}

// OrderQuotationItem defines one order line item relevant to quotation package building.
type OrderQuotationItem struct {
	// SKU defines the product SKU used to resolve shipping attributes.
	SKU string
	// Quantity defines the ordered quantity.
	Quantity int
}

// OrderProductSource resolves product shipping attributes required for quotation package building.
type OrderProductSource interface {
	// GetShippingAttributes resolves shipping dimension and packing attributes for one SKU.
	GetShippingAttributes(ctx context.Context, sku string) (*ProductShippingAttributes, error)
}

// ProductShippingAttributes defines product physical attributes used for package building.
type ProductShippingAttributes struct {
	// SKU defines the product SKU.
	SKU string
	// WeightKG defines real weight in kilograms.
	WeightKG float64
	// HeightCM defines height in centimeters.
	HeightCM float64
	// WidthCM defines width in centimeters.
	WidthCM float64
	// LengthCM defines length (depth) in centimeters.
	LengthCM float64
	// Price defines the unit price used for declared value calculations.
	Price float64
	// Overlapped reports whether this product should be packed inside another box.
	// When true the product is treated as an overlapped/nested item; when false it is a standalone box.
	Overlapped bool
	// Valid reports whether all required dimension fields are present and non-zero.
	Valid bool
}
