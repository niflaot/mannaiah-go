package product

import (
	"context"
	"errors"

	productdomain "mannaiah/module/products/domain/product"
)

var (
	// ErrNotFound is returned when product records are missing.
	ErrNotFound = errors.New("product not found")
	// ErrDuplicateSKU is returned when a SKU already exists.
	ErrDuplicateSKU = errors.New("product sku already exists")
)

// Repository defines product persistence contracts.
type Repository interface {
	// EnsureSchema ensures storage schema availability.
	EnsureSchema(ctx context.Context) error
	// Create persists a new product.
	Create(ctx context.Context, product *productdomain.Product) error
	// GetByID retrieves products by ID.
	GetByID(ctx context.Context, id string) (*productdomain.Product, error)
	// List retrieves non-deleted products.
	List(ctx context.Context) ([]productdomain.Product, error)
	// Update persists product updates.
	Update(ctx context.Context, product *productdomain.Product) error
	// Delete soft-deletes products by ID.
	Delete(ctx context.Context, id string) error
}
