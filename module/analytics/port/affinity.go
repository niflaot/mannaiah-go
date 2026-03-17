package port

import (
	"context"

	"mannaiah/module/analytics/domain"
)

// AffinityStore defines ClickHouse-backed affinity scoring behavior.
type AffinityStore interface {
	// GetTagAffinity retrieves ranked tag affinity scores for one contact.
	GetTagAffinity(ctx context.Context, contactID string, limit int, minScore float64) ([]domain.TagAffinity, error)
	// GetCategoryAffinity retrieves ranked category affinity scores for one contact.
	GetCategoryAffinity(ctx context.Context, contactID string, limit int, minScore float64) ([]domain.CategoryAffinity, error)
	// GetVariationAffinity retrieves ranked variation affinity scores for one contact.
	GetVariationAffinity(ctx context.Context, contactID string, limit int, minScore float64) ([]domain.VariationAffinity, error)
	// GetProfile assembles a full affinity profile for one contact.
	GetProfile(ctx context.Context, contactID string, limit int, minScore float64) (*domain.AffinityProfile, error)
	// RefreshTagMV truncates and repopulates the tag_affinity_mv table.
	RefreshTagMV(ctx context.Context) error
	// RefreshCategoryMV truncates and repopulates the category_affinity_mv table.
	RefreshCategoryMV(ctx context.Context) error
	// RefreshVariationMV truncates and repopulates the variation_affinity_mv table.
	RefreshVariationMV(ctx context.Context) error
}
