package port

import (
	"context"
	"time"
)

// WooOrderItem defines WooCommerce order item values.
type WooOrderItem struct {
	// SKU defines order item SKU values.
	SKU string
	// Name defines order item name values.
	Name string
	// Quantity defines order item quantity values.
	Quantity int
	// Value defines item monetary value values.
	Value float64
}

// WooOrderShippingCharge defines WooCommerce order shipping charge values.
type WooOrderShippingCharge struct {
	// MethodID defines shipping method identifier values.
	MethodID string
	// MethodTitle defines shipping method title values.
	MethodTitle string
	// Price defines shipping price values.
	Price float64
}

// WooOrderComment defines WooCommerce order comment values.
type WooOrderComment struct {
	// Author defines comment author values.
	Author string
	// Description defines comment text values.
	Description string
	// Internal reports whether comments are internal-only.
	Internal bool
	// OccurredAt defines comment timestamps.
	OccurredAt time.Time
}

// WooOrder defines order data retrieved from WooCommerce.
type WooOrder struct {
	// ID defines WooCommerce order identifiers.
	ID int
	// BillingEmail defines order billing email values.
	BillingEmail string
	// BillingFirstName defines order billing first-name values.
	BillingFirstName string
	// BillingLastName defines order billing last-name values.
	BillingLastName string
	// BillingCompany defines order billing company values.
	BillingCompany string
	// BillingPhone defines order billing phone values.
	BillingPhone string
	// BillingAddress1 defines order billing address line 1 values.
	BillingAddress1 string
	// BillingAddress2 defines order billing address line 2 values.
	BillingAddress2 string
	// BillingCity defines order billing city values.
	BillingCity string
	// CreatedAt defines order creation timestamps.
	CreatedAt time.Time
	// Status defines order status values.
	Status string
	// BillingAddressLine1 defines billing address line 1 values.
	BillingAddressLine1 string
	// BillingAddressLine2 defines billing address line 2 values.
	BillingAddressLine2 string
	// BillingCityCode defines billing city-code values.
	BillingCityCode string
	// BillingPhoneNormalized defines billing phone values.
	BillingPhoneNormalized string
	// ShippingFirstName defines shipping first-name values.
	ShippingFirstName string
	// ShippingLastName defines shipping last-name values.
	ShippingLastName string
	// ShippingCompany defines shipping company values.
	ShippingCompany string
	// ShippingPhone defines shipping phone values.
	ShippingPhone string
	// ShippingAddressLine1 defines shipping address line 1 values.
	ShippingAddressLine1 string
	// ShippingAddressLine2 defines shipping address line 2 values.
	ShippingAddressLine2 string
	// ShippingCityCode defines shipping city-code values.
	ShippingCityCode string
	// PaymentMethod defines order payment method values.
	PaymentMethod string
	// Items defines order item values.
	Items []WooOrderItem
	// ShippingCharges defines order shipping charge values.
	ShippingCharges []WooOrderShippingCharge
	// Comments defines order comment values.
	Comments []WooOrderComment
	// Metadata defines order metadata values keyed by metadata key.
	Metadata map[string]string
}

// OrderSource defines WooCommerce order retrieval behavior.
type OrderSource interface {
	// Validate verifies source connectivity and credentials.
	Validate(ctx context.Context) error
	// ListOrders retrieves paginated order values and reports whether additional pages exist.
	ListOrders(ctx context.Context, page int, pageSize int) (orders []WooOrder, hasNext bool, err error)
}
