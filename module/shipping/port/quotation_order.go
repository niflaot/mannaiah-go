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
	// CollectOnDeliveryAmount defines COD amount resolved from the order payment method.
	// It must be zero when the order is not cash-on-delivery.
	CollectOnDeliveryAmount float64
	// RecipientName defines recipient display-name values resolved from the order context.
	RecipientName string
	// RecipientID defines optional recipient identification values resolved from the order context.
	RecipientID string
	// RecipientIDType defines optional recipient identification-type values resolved from the order context.
	RecipientIDType string
	// RecipientAddressLine defines recipient address-line values resolved from the order shipping address.
	RecipientAddressLine string
	// RecipientPhone defines recipient phone values resolved from the order shipping address.
	RecipientPhone string
	// RecipientEmail defines optional recipient email values resolved from the order context.
	RecipientEmail string
	// Items defines the line items belonging to the order.
	Items []OrderQuotationItem
}

// OrderQuotationItem defines one order line item relevant to quotation package building.
type OrderQuotationItem struct {
	// SKU defines the product SKU used to resolve shipping attributes.
	// May be empty when the item was resolved by product name; use ProductID as fallback.
	SKU string
	// ProductID defines the resolved internal product identifier.
	// Populated when the order item was matched by alternate name rather than SKU.
	ProductID string
	// Quantity defines the ordered quantity.
	Quantity int
}

// OrderProductSource resolves product shipping attributes required for quotation package building.
type OrderProductSource interface {
	// GetShippingAttributes resolves shipping dimension and packing attributes for one SKU.
	GetShippingAttributes(ctx context.Context, sku string) (*ProductShippingAttributes, error)
	// GetShippingAttributesByID resolves shipping dimension and packing attributes by internal product ID.
	// Used as fallback when an order item has no SKU but has a resolved ProductID.
	GetShippingAttributesByID(ctx context.Context, productID string) (*ProductShippingAttributes, error)
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
