package port

import "context"

// MainstreamOrderUpdateCommand defines mainstream-origin order update payload values for WooCommerce.
type MainstreamOrderUpdateCommand struct {
	// Identifier defines external WooCommerce order identifiers.
	Identifier string
	// Status defines optional WooCommerce status values (for example, processing/completed).
	Status string
	// ShippingAddress defines explicit shipping-address values.
	ShippingAddress *OrderSyncShippingAddress
	// ShippingCharges defines shipping charge values.
	ShippingCharges []OrderSyncShippingCharge
	// Items defines order item values.
	Items []OrderSyncItem
}

// OrderDestination defines WooCommerce order update behavior for mainstream-origin changes.
type OrderDestination interface {
	// Validate verifies destination connectivity and credentials.
	Validate(ctx context.Context) error
	// UpdateOrderFromMainstream updates WooCommerce orders from mainstream-origin payload values.
	UpdateOrderFromMainstream(ctx context.Context, command MainstreamOrderUpdateCommand) error
}
