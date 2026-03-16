package category

import (
	"context"
	"errors"

	categorydomain "mannaiah/module/products/domain/category"
	productdomain "mannaiah/module/products/domain/product"
)

var (
	// ErrNotFound is returned when a category record is missing.
	ErrNotFound = errors.New("category not found")
	// ErrDuplicateSlug is returned when a slug already exists.
	ErrDuplicateSlug = errors.New("category slug already exists")
	// ErrHasChildren is returned when a category has children and cannot be deleted.
	ErrHasChildren = errors.New("category has children and cannot be deleted")
)

// ListProductsQuery defines parameters for product listing within a category.
type ListProductsQuery struct {
	// CategoryID defines the target category identifier.
	CategoryID string
	// Page defines the 1-based page number.
	Page int
	// PageSize defines the maximum number of results per page.
	PageSize int
}

// ListProductsResult defines paginated product listing results.
type ListProductsResult struct {
	// Items defines the product listing results.
	Items []*productdomain.Product
	// Total defines the total number of matching products.
	Total int64
	// Page defines the current page number.
	Page int
	// PageSize defines the page size used for this result.
	PageSize int
}

// Repository defines category persistence behavior.
type Repository interface {
	// EnsureSchema ensures storage schema availability.
	EnsureSchema(ctx context.Context) error
	// Create persists a new category.
	Create(ctx context.Context, cat *categorydomain.Category) error
	// GetByID retrieves a category by ID.
	GetByID(ctx context.Context, id string) (*categorydomain.Category, error)
	// GetBySlug retrieves a category by slug.
	GetBySlug(ctx context.Context, slug string) (*categorydomain.Category, error)
	// Tree retrieves all root-level (non-deleted) categories.
	Tree(ctx context.Context) ([]*categorydomain.Category, error)
	// ListChildren retrieves direct children of a category.
	ListChildren(ctx context.Context, parentID string) ([]*categorydomain.Category, error)
	// Update persists category updates.
	Update(ctx context.Context, cat *categorydomain.Category) error
	// Delete soft-deletes a category by ID.
	Delete(ctx context.Context, id string) error
	// ListProducts resolves and returns paginated products for a category.
	ListProducts(ctx context.Context, q ListProductsQuery) (*ListProductsResult, error)
}
