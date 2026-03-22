package domain

const (
	// BaseTagModeAny returns products that carry at least one of the specified base tags (union).
	// This is the default when BaseTagMode is empty.
	BaseTagModeAny = "any"
	// BaseTagModeAll returns only products that carry every specified base tag (intersection).
	BaseTagModeAll = "all"
)

// RecommendationQuery defines parameters for per-contact product recommendation resolution.
type RecommendationQuery struct {
	// BaseTag is a single product base tag for backward compatibility.
	// It is merged into BaseTags during Normalize — prefer BaseTags for new callers.
	BaseTag string
	// BaseTags is a list of product base tags. BaseTagMode controls whether candidates
	// must match any tag (union) or all tags (intersection).
	// At least one of BaseTag or BaseTags must be non-empty unless PinnedProductIDs is set.
	BaseTags []string
	// BaseTagMode controls multi-tag matching semantics.
	// "any" (default) — union: products with at least one matching tag.
	// "all"            — intersection: products that carry every tag in BaseTags.
	BaseTagMode string
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

	// Merge BaseTag into BaseTags for unified processing.
	if q.BaseTag != "" {
		found := false
		for _, t := range q.BaseTags {
			if t == q.BaseTag {
				found = true
				break
			}
		}
		if !found {
			q.BaseTags = append([]string{q.BaseTag}, q.BaseTags...)
		}
	}

	// Deduplicate BaseTags preserving order.
	if len(q.BaseTags) > 1 {
		seen := make(map[string]struct{}, len(q.BaseTags))
		deduped := make([]string, 0, len(q.BaseTags))
		for _, t := range q.BaseTags {
			if _, ok := seen[t]; !ok {
				seen[t] = struct{}{}
				deduped = append(deduped, t)
			}
		}
		q.BaseTags = deduped
	}

	if q.BaseTagMode != BaseTagModeAll {
		q.BaseTagMode = BaseTagModeAny
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
	// URL is the realm-scoped product detail URL when available.
	URL string `json:"url,omitempty"`
}
