package domain

import "time"

// ProductBlock defines a product recommendation block stored with a campaign.
// Each block is rendered into a named product list available in the template context.
type ProductBlock struct {
	// ID is the block identifier used as the key in the template Products map.
	ID string `json:"id"`
	// BaseTag is a single base tag shorthand (backward compatible). Merged into BaseTags.
	BaseTag string `json:"baseTag"`
	// BaseTags lists product base tags. BaseTagMode controls union vs intersection matching.
	BaseTags []string `json:"baseTags,omitempty"`
	// BaseTagMode controls how BaseTags are matched:
	// "any" (default) — products with at least one tag (union).
	// "all"            — products that carry every tag (intersection).
	BaseTagMode string `json:"baseTagMode,omitempty"`
	// UseAffinity enables contact-affinity-driven filtering when true.
	UseAffinity bool `json:"useAffinity"`
	// AffinityMinScorePct is the minimum relative affinity score threshold in [0, 100].
	AffinityMinScorePct float64 `json:"affinityMinScorePct"`
	// CategoryID optionally restricts dynamic candidates to one product category identifier.
	CategoryID string `json:"categoryId"`
	// Realm identifies which product datasheet and gallery to use for name and image resolution.
	Realm string `json:"realm"`
	// Limit is the maximum number of products to return (clamped to [1, 10]).
	Limit int `json:"limit"`
	// PinnedProductIDs lists product IDs that are always included first in results,
	// regardless of base tag or affinity. Pinned products are loaded by ID and
	// prepended before any dynamically ranked candidates.
	PinnedProductIDs []string `json:"pinnedProductIds,omitempty"`
	// ExcludeProductIDs lists product IDs that must never appear in results.
	ExcludeProductIDs []string `json:"excludeProductIds,omitempty"`
	// FilterVariationIDs restricts candidates to products that carry at least one of
	// these variation IDs. Optional — when empty, no variation filtering is applied.
	FilterVariationIDs []string `json:"filterVariationIds,omitempty"`
	// PreferVariationIDs biases gallery image selection toward images linked to these
	// variation IDs. Optional — when empty, the first realm-visible image is used.
	PreferVariationIDs []string `json:"preferVariationIds,omitempty"`
}

// TemplateProduct defines one product entry available inside the campaign template context.
type TemplateProduct struct {
	// ID is the product identifier.
	ID string
	// Name is the realm-resolved display name.
	Name string
	// Price is the product price (zero if unset).
	Price float64
	// ImageURL is the public URL of the first realm-matched gallery image.
	ImageURL string
	// URL is the realm-scoped product detail URL when available.
	URL string
}

// ContactTemplateData defines the per-contact data available inside the campaign template context.
type ContactTemplateData struct {
	// Name is the contact short display name (first-name preference).
	Name string
	// FullName is the complete contact display name.
	FullName string
	// FirstName is the first word of the contact name before the first space.
	FirstName string
	// Email is the contact email address.
	Email string
	// LastSaleDate is the date of the contact's most recent purchase, or nil if unknown.
	LastSaleDate *time.Time
}

// TemplateContext defines the data available to campaign template renderers.
type TemplateContext struct {
	// Contact contains per-contact personalization data.
	Contact ContactTemplateData
	// Custom contains arbitrary campaign-level custom variable values.
	Custom map[string]string
	// Products maps product block IDs to their resolved product lists.
	Products map[string][]TemplateProduct
}
