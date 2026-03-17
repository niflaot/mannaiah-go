package products

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
	ordersport "mannaiah/module/orders/port"
)

var (
	// ErrNilDB is returned when DB dependencies are nil.
	ErrNilDB = errors.New("orders products resolver db must not be nil")
)

// Resolver defines product lookup behavior for order-item resolution.
type Resolver struct {
	// db defines query dependencies.
	db *gorm.DB
}

type productRow struct {
	ID string `gorm:"column:id"`
}

var (
	// _ ensures Resolver satisfies product-resolver contracts.
	_ ordersport.ProductResolver = (*Resolver)(nil)
)

// NewResolver creates product resolver adapters.
func NewResolver(db *gorm.DB) (*Resolver, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	return &Resolver{db: db}, nil
}

// Resolve resolves products by SKU first, variant SKU second, and alternate-name fallback third.
func (r *Resolver) Resolve(ctx context.Context, sku string, alternateName string) (*ordersport.ProductResolution, error) {
	sku = strings.TrimSpace(sku)
	alternateName = strings.TrimSpace(alternateName)

	if sku != "" {
		product, err := r.findBySKU(ctx, sku)
		if err != nil {
			return nil, err
		}
		if product != nil {
			return &ordersport.ProductResolution{ProductID: product.ID, MatchedBy: "sku"}, nil
		}

		product, err = r.findByVariantSKU(ctx, sku)
		if err != nil {
			return nil, err
		}
		if product != nil {
			return &ordersport.ProductResolution{ProductID: product.ID, MatchedBy: "variant_sku"}, nil
		}
	}

	if alternateName != "" {
		product, err := r.findByAlternateName(ctx, alternateName)
		if err != nil {
			return nil, err
		}
		if product != nil {
			return &ordersport.ProductResolution{ProductID: product.ID, MatchedBy: "alternate_name"}, nil
		}
	}

	return nil, nil
}

// findBySKU resolves products by SKU values.
func (r *Resolver) findBySKU(ctx context.Context, sku string) (*productRow, error) {
	var row productRow
	err := r.db.WithContext(ctx).Table("products").Select("id").Where("sku = ?", strings.TrimSpace(sku)).Limit(1).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &row, nil
}

// findByVariantSKU resolves parent products by variant-level SKU values.
func (r *Resolver) findByVariantSKU(ctx context.Context, sku string) (*productRow, error) {
	var row productRow
	err := r.db.WithContext(ctx).
		Table("product_variants").
		Select("product_variants.product_id as id").
		Where("product_variants.sku = ?", strings.TrimSpace(sku)).
		Order("product_variants.id asc").
		Limit(1).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &row, nil
}

// findByAlternateName resolves products by datasheet-name values.
func (r *Resolver) findByAlternateName(ctx context.Context, name string) (*productRow, error) {
	var row productRow
	err := r.db.WithContext(ctx).
		Table("product_datasheets").
		Select("product_datasheets.product_id as id").
		Where("LOWER(product_datasheets.name) = LOWER(?)", strings.TrimSpace(name)).
		Order("product_datasheets.id asc").
		Limit(1).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &row, nil
}
