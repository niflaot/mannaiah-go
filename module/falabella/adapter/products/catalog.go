package products

import (
	"context"
	"errors"
	"fmt"

	productdomain "mannaiah/module/products/domain/product"
	"mannaiah/module/falabella/port"
)

var (
	// ErrNilService is returned when product-service dependencies are nil.
	ErrNilService = errors.New("products service must not be nil")
)

// service defines product-service behavior required by this adapter.
type service interface {
	// Get retrieves products by identifier.
	Get(ctx context.Context, id string) (*productdomain.Product, error)
	// List retrieves all products.
	List(ctx context.Context) ([]productdomain.Product, error)
}

// Catalog defines product-catalog adapters backed by module/products services.
type Catalog struct {
	// service defines product-service dependencies.
	service service
}

var (
	// _ ensures Catalog satisfies Falabella product-catalog ports.
	_ port.ProductCatalog = (*Catalog)(nil)
)

// NewCatalog creates Falabella product-catalog adapters.
func NewCatalog(service service) (*Catalog, error) {
	if service == nil {
		return nil, ErrNilService
	}

	return &Catalog{service: service}, nil
}

// GetProduct retrieves mapped catalog products by identifier.
func (c *Catalog) GetProduct(ctx context.Context, id string) (*port.CatalogProduct, error) {
	entity, err := c.service.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get product from products module: %w", err)
	}

	return mapProduct(entity), nil
}

// ListProducts retrieves mapped catalog products.
func (c *Catalog) ListProducts(ctx context.Context) ([]port.CatalogProduct, error) {
	entities, err := c.service.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list products from products module: %w", err)
	}

	mapped := make([]port.CatalogProduct, 0, len(entities))
	for _, entity := range entities {
		copied := entity
		mapped = append(mapped, *mapProduct(&copied))
	}

	return mapped, nil
}

// mapProduct maps product-domain entities into Falabella catalog-product values.
func mapProduct(entity *productdomain.Product) *port.CatalogProduct {
	if entity == nil {
		return nil
	}

	datasheets := make([]port.CatalogDatasheet, 0, len(entity.Datasheets))
	for _, item := range entity.Datasheets {
		attributes := make(map[string]any, len(item.Attributes))
		for key, value := range item.Attributes {
			attributes[key] = value
		}
		datasheets = append(datasheets, port.CatalogDatasheet{
			Realm:       item.Realm,
			Name:        item.Name,
			Description: item.Description,
			Attributes:  attributes,
		})
	}

	return &port.CatalogProduct{
		ID:         entity.ID,
		SKU:        entity.SKU,
		Datasheets: datasheets,
	}
}
