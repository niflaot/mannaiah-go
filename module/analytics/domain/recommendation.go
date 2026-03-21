package domain

// RecommendationQuery defines parameters for per-contact product recommendation resolution.
type RecommendationQuery struct {
	// BaseTag is a required product tag; only products with this tag are candidates.
	BaseTag string
	// UseContactAffinity enables affinity-driven filtering when true.
	// When true, the resolver expands the contact's top affinity tags via tag_correlations
	// and further filters to candidates that share at least one correlated tag.
	UseContactAffinity bool
	// AffinityMinScorePct is the minimum relative affinity score threshold in [0, 100].
	// Only affinity tags at or above this percentile contribute to the expanded tag set.
	AffinityMinScorePct float64
	// CategoryID optionally restricts candidates to one product category identifier.
	CategoryID string
	// Realm identifies which product datasheet and gallery to use for name and image resolution.
	// Defaults to "default" when empty.
	Realm string
	// Limit is the maximum number of products to return (clamped to [1, 10]).
	Limit int
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
	// Price is the product price (zero if unset).
	Price float64 `json:"price"`
	// ImageURL is the public URL of the first realm-matched gallery image.
	ImageURL string `json:"imageUrl"`
}
