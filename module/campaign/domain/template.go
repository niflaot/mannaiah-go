package domain

import "time"

// ProductBlock defines a product recommendation block stored with a campaign.
// Each block is rendered into a named product list available in the template context.
type ProductBlock struct {
	// ID is the block identifier used as the key in the template Products map.
	ID string `json:"id"`
	// BaseTag is the required product base tag; only products with this tag are candidates.
	BaseTag string `json:"baseTag"`
	// UseAffinity enables contact-affinity-driven filtering when true.
	UseAffinity bool `json:"useAffinity"`
	// AffinityMinScorePct is the minimum relative affinity score threshold in [0, 100].
	AffinityMinScorePct float64 `json:"affinityMinScorePct"`
	// CategoryID optionally restricts candidates to one product category identifier.
	CategoryID string `json:"categoryId"`
	// Realm identifies which product datasheet and gallery to use for name and image resolution.
	Realm string `json:"realm"`
	// Limit is the maximum number of products to return (clamped to [1, 10]).
	Limit int `json:"limit"`
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
}

// ContactTemplateData defines the per-contact data available inside the campaign template context.
type ContactTemplateData struct {
	// Name is the contact display name.
	Name string
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
