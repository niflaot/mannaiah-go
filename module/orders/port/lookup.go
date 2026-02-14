package port

import (
	"context"
	"errors"
)

var (
	// ErrCustomerNotFound is returned when customers are unavailable for order linkage.
	ErrCustomerNotFound = errors.New("order customer not found")
)

// Customer defines customer billing-address values required by orders.
type Customer struct {
	// ID defines customer identifiers.
	ID string
	// Address defines billing address line 1 values.
	Address string
	// AddressExtra defines billing address line 2 values.
	AddressExtra string
	// Phone defines billing phone values.
	Phone string
	// CityCode defines billing city-code values.
	CityCode string
}

// ProductResolution defines product lookup result values.
type ProductResolution struct {
	// ProductID defines resolved product identifiers.
	ProductID string
	// MatchedBy defines resolution source values (`sku` or `alternate_name`).
	MatchedBy string
}

// CustomerSource defines customer lookup behavior required by orders.
type CustomerSource interface {
	// GetByID resolves customer values by identifiers.
	GetByID(ctx context.Context, id string) (*Customer, error)
}

// ProductResolver defines product lookup behavior for order-item resolution.
type ProductResolver interface {
	// Resolve resolves products by SKU first and alternate-name fallback second.
	Resolve(ctx context.Context, sku string, alternateName string) (*ProductResolution, error)
}
