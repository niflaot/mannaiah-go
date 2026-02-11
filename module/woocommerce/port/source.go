package port

import "context"

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
