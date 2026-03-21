package port

import (
	"context"

	"mannaiah/module/campaign/domain"
)

// AffinityProductProvider defines product recommendation fetch behavior for campaign rendering.
type AffinityProductProvider interface {
	// GetProducts returns recommended products for one contact and one product block configuration.
	// Returns nil, nil when no products match the query (fail-open).
	GetProducts(ctx context.Context, contactID string, block domain.ProductBlock) ([]domain.TemplateProduct, error)
}

// NoopAffinityProductProvider returns nil products for all queries.
type NoopAffinityProductProvider struct{}

// GetProducts returns nil, nil.
func (NoopAffinityProductProvider) GetProducts(_ context.Context, _ string, _ domain.ProductBlock) ([]domain.TemplateProduct, error) {
	return nil, nil
}
