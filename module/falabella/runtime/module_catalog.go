package runtime

import (
	"context"
	"errors"
	"fmt"

	productsyncservice "mannaiah/module/falabella/application/productsync/service"
	"mannaiah/module/falabella/port"
)

var (
	// ErrProductCatalogNotConfigured is returned when product-sync endpoints are used without catalog dependencies.
	ErrProductCatalogNotConfigured = errors.New("falabella product catalog is not configured")
)

// failingCatalog defines unavailable product-catalog behavior.
type failingCatalog struct {
	// err defines startup wiring errors.
	err error
}

// GetProduct returns startup wiring failures.
func (f failingCatalog) GetProduct(ctx context.Context, id string) (*port.CatalogProduct, error) {
	return nil, f.err
}

// ListProducts returns startup wiring failures.
func (f failingCatalog) ListProducts(ctx context.Context) ([]port.CatalogProduct, error) {
	return nil, f.err
}

// resolveCatalog resolves optional product-catalog dependencies.
func resolveCatalog(catalogs ...port.ProductCatalog) port.ProductCatalog {
	if len(catalogs) == 0 || catalogs[0] == nil {
		return failingCatalog{err: fmt.Errorf("%w: %w", productsyncservice.ErrIntegrationUnavailable, ErrProductCatalogNotConfigured)}
	}

	return catalogs[0]
}

var (
	// _ ensures failingCatalog satisfies product-catalog contracts.
	_ port.ProductCatalog = (*failingCatalog)(nil)
)
