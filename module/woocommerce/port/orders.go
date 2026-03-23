package port

import (
	"context"
	"time"
)

// OrderSyncItem defines order item sync payload values.
type OrderSyncItem struct {
	// SKU defines order item SKU values.
	SKU string
	// Name defines order item display names.
	Name string
	// Quantity defines item quantity values.
	Quantity int
	// Value defines item monetary value values.
	Value float64
}

// OrderSyncShippingAddress defines order shipping-address sync values.
type OrderSyncShippingAddress struct {
	// Address defines shipping address line 1 values.
	Address string
	// Address2 defines shipping address line 2 values.
	Address2 string
	// Phone defines shipping phone values.
	Phone string
	// CityCode defines shipping city-code values.
	CityCode string
}

// OrderSyncShippingCharge defines order shipping charge sync values.
type OrderSyncShippingCharge struct {
	// MethodID defines shipping method identifier values.
	MethodID string
	// MethodTitle defines shipping method title values.
	MethodTitle string
	// Price defines shipping price values.
	Price float64
}

// OrderSyncComment defines order comment sync values.
type OrderSyncComment struct {
	// Author defines comment author values.
	Author string
	// Comment defines comment text values.
	Comment string
	// Internal reports whether comments are internal-only.
	Internal bool
	// OccurredAt defines comment timestamps.
	OccurredAt time.Time
}

// OrderSyncCommand defines order upsert payload values produced by WooCommerce syncs.
type OrderSyncCommand struct {
	// Identifier defines external order identifiers.
	Identifier string
	// Realm defines order realm values.
	Realm string
	// Status defines source status values.
	Status string
	// PaymentMethod defines order payment method values.
	PaymentMethod string
	// CreatedAt defines source order creation timestamps.
	CreatedAt *time.Time
	// Contact defines contact-sync payload values used when target contacts do not exist.
	Contact ContactSyncCommand
	// ShippingAddress defines optional custom shipping-address values.
	ShippingAddress *OrderSyncShippingAddress
	// ShippingCharges defines order shipping charge values.
	ShippingCharges []OrderSyncShippingCharge
	// Items defines order item values.
	Items []OrderSyncItem
	// Metadata defines order metadata values.
	Metadata map[string]string
	// Comments defines order comment values.
	Comments []OrderSyncComment
}

// OrderSyncTarget defines order upsert behavior required by WooCommerce sync services.
type OrderSyncTarget interface {
	// UpsertByIdentifier creates or updates orders keyed by realm+identifier and reports upsert outcomes.
	UpsertByIdentifier(ctx context.Context, command OrderSyncCommand) (outcome UpsertOutcome, err error)
}
