package runtime

import (
	"context"

	categoryapplication "mannaiah/module/products/application/category"
	categorydomain "mannaiah/module/products/domain/category"
	productdomain "mannaiah/module/products/domain/product"
)

const (
	// storefrontNavigationPageSize defines the category product page size used for full navigation regeneration.
	storefrontNavigationPageSize = 100000
)

// storefrontNavigationSource adapts category application services into storefront navigation data sources.
type storefrontNavigationSource struct {
	// categoryService defines category query dependencies.
	categoryService categoryapplication.Service
}

// Tree returns all root categories ordered from oldest to newest.
func (s storefrontNavigationSource) Tree(ctx context.Context) ([]*categorydomain.Category, error) {
	return s.categoryService.Tree(ctx)
}

// Children returns all direct child categories ordered from oldest to newest.
func (s storefrontNavigationSource) Children(ctx context.Context, parentID string) ([]*categorydomain.Category, error) {
	return s.categoryService.Children(ctx, parentID)
}

// ListProducts returns all products visible under the provided category ordered from oldest to newest.
func (s storefrontNavigationSource) ListProducts(ctx context.Context, categoryID string) ([]*productdomain.Product, error) {
	result, err := s.categoryService.ListProducts(ctx, categoryapplication.ListProductsQuery{
		CategoryID: categoryID,
		Page:       1,
		PageSize:   storefrontNavigationPageSize,
	})
	if err != nil {
		return nil, err
	}

	return result.Items, nil
}
