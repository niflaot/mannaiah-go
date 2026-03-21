package domain

// RecommendationQuery defines parameters for per-contact product recommendation resolution.
type RecommendationQuery struct {
	// BaseTag is the product base tag; only products with this tag are dynamic candidates.
	// May be empty when PinnedProductIDs is non-empty (pinned-only mode).
	BaseTag string
	// UseContactAffinity enables affinity-driven filtering when true.
	UseContactAffinity bool
	// AffinityMinScorePct is the minimum relative affinity score threshold in [0, 100].
	AffinityMinScorePct float64
	// CategoryID optionally restricts candidates to one product category identifier.
	CategoryID string
	// Realm identifies which product datasheet and gallery to use for name and image resolution.
	// Defaults to "default" when empty.
	Realm string
	// Limit is the maximum number of products to return (clamped to [1, 10]).
	Limit int
	// PinnedProductIDs lists product IDs that are always included first in the result,
	// regardless of base tag or affinity.
	PinnedProductIDs []string
	// ExcludeProductIDs lists product IDs that must never appear in results.
	ExcludeProductIDs []string
	// FilterVariationIDs restricts candidates to products that carry at least one of
	// these variation IDs (e.g. only show products available in black).
	// Optional — when empty, no variation filtering is applied.
	FilterVariationIDs []string
	// PreferVariationIDs biases gallery image selection toward images linked to these
	// variation IDs (e.g. prefer the black-variant photo). Falls back to the first
	// realm-visible image when no variation-specific image is found.
	// Optional — when empty, the first realm-visible image is used.
	PreferVariationIDs []string
}

// Normalize canonicalizes query values before resolution.
func (q *RecommendationQuery) Normalize() {
	if q == nil {
		return
	}
	if q.Realm == "" {
		q.Realm = "default"
	}
	if q.Limit <= 0 {
		q.Limit = 3
	}
	if q.Limit > 10 {
		q.Limit = 10
	}
}

// RecommendedProduct defines one product recommendation result.
type RecommendedProduct struct {
	// ID is the product identifier.
	ID string `json:"id"`
	// Name is the realm-resolved display name.
	Name string `json:"name"`
	// Price is the realm-resolved price from the product datasheet attributes.
	Price float64 `json:"price"`
	// ImageURL is the public URL of the first realm-matched gallery image.
	ImageURL string `json:"imageUrl"`
}
