package port

import (
	"context"
	"time"

	ordersdomain "mannaiah/module/orders/domain"
)

// OrderSyncItemCommand defines one normalized Shopify order item.
type OrderSyncItemCommand struct {
	// SKU defines SKU values.
	SKU string
	// AlternateName defines fallback item-name values.
	AlternateName string
	// Quantity defines ordered quantity values.
	Quantity int
	// Value defines unit-value values.
	Value float64
}

// OrderSyncShippingAddressCommand defines one normalized shipping address payload.
type OrderSyncShippingAddressCommand struct {
	// Address defines address line 1 values.
	Address string
	// Address2 defines address line 2 values.
	Address2 string
	// Phone defines phone values.
	Phone string
	// CityCode defines normalized city-code values.
	CityCode string
}

// OrderSyncShippingChargeCommand defines one normalized shipping charge payload.
type OrderSyncShippingChargeCommand struct {
	// MethodID defines shipping method identifiers.
	MethodID string
	// MethodTitle defines shipping method display titles.
	MethodTitle string
	// Price defines shipping charge amounts.
	Price float64
}

// OrderSyncAppliedCouponCommand defines one normalized applied coupon payload.
type OrderSyncAppliedCouponCommand struct {
	// Code defines coupon code values.
	Code string
	// DiscountType defines discount type values.
	DiscountType string
	// DiscountAmount defines discount amount values.
	DiscountAmount float64
	// AppliedAt defines coupon application timestamps.
	AppliedAt *time.Time
}

// OrderSyncCommand defines normalized order upsert payload values.
type OrderSyncCommand struct {
	// ShopDomain defines the source Shopify store domain.
	ShopDomain string
	// ShopifyID defines the source Shopify order identifier.
	ShopifyID string
	// Identifier defines public/external order identifiers used in Mannaiah.
	Identifier string
	// Realm defines order realm values.
	Realm string
	// ContactID defines resolved mainstream contact identifiers.
	ContactID string
	// Items defines normalized line-item values.
	Items []OrderSyncItemCommand
	// Status defines mapped mainstream order status values.
	Status ordersdomain.Status
	// StatusDescription defines status transition descriptions.
	StatusDescription string
	// ShippingAddress defines normalized shipping address values.
	ShippingAddress *OrderSyncShippingAddressCommand
	// ShippingCharges defines normalized shipping charge values.
	ShippingCharges []OrderSyncShippingChargeCommand
	// AppliedCoupons defines normalized coupon values.
	AppliedCoupons []OrderSyncAppliedCouponCommand
	// PaymentMethod defines payment method labels.
	PaymentMethod string
	// Metadata defines normalized metadata values.
	Metadata map[string]string
	// CreatedAt defines optional source creation timestamps.
	CreatedAt *time.Time
	// Source defines mutation source values.
	Source string
}

// OrderSyncTarget defines mainstream order upsert behavior.
type OrderSyncTarget interface {
	// UpsertOrder creates or updates one mainstream order from Shopify values.
	UpsertOrder(ctx context.Context, command OrderSyncCommand) (*ordersdomain.Order, error)
}
