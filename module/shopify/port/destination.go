package port

import (
	"context"

	ordersport "mannaiah/module/orders/port"
)

// ShopifyOrderDestination defines Shopify order mutation behavior.
type ShopifyOrderDestination interface {
	// ApplyOrderUpdate applies safe Mannaiah-origin order edits to Shopify.
	ApplyOrderUpdate(ctx context.Context, shopifyOrderID string, payload ordersport.OrderEventPayload, variantResolver ShopifyVariantResolver) error
	// CancelOrder cancels one Shopify order without customer notification.
	CancelOrder(ctx context.Context, shopifyOrderID string, reason string) error
}

// ShopifyVariantResolver defines Mannaiah-product to Shopify-variant lookup behavior.
type ShopifyVariantResolver interface {
	// ResolveVariantID resolves one Shopify variant ID for a Mannaiah product ID.
	ResolveVariantID(ctx context.Context, productID string) (string, error)
}

// ShopifyFulfillmentDestination defines Shopify fulfillment mutation behavior.
type ShopifyFulfillmentDestination interface {
	// FulfillOrder creates one Shopify fulfillment with tracking data.
	FulfillOrder(ctx context.Context, input ShopifyFulfillOrderInput) (string, error)
	// CancelFulfillment cancels one Shopify fulfillment.
	CancelFulfillment(ctx context.Context, fulfillmentID string) error
}

// ShopifyFulfillOrderInput defines Shopify fulfillment payload values.
type ShopifyFulfillOrderInput struct {
	// ShopifyOrderID defines Shopify order identifiers.
	ShopifyOrderID string
	// TrackingNumber defines shipment tracking numbers.
	TrackingNumber string
	// TrackingCompany defines shipment carrier names.
	TrackingCompany string
	// TrackingURL defines optional shipment tracking URLs.
	TrackingURL string
	// NotifyCustomer reports whether Shopify should notify the customer.
	NotifyCustomer bool
}
