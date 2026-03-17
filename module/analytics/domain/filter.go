package domain

// AffinityTagFilter defines a tag affinity segment filter constraint.
type AffinityTagFilter struct {
	// Tag is the product tag to match.
	Tag string
	// MinScore is the minimum affinity score threshold (inclusive).
	MinScore float64
}

// AffinityCategoryFilter defines a category affinity segment filter constraint.
type AffinityCategoryFilter struct {
	// CategoryID is the product category identifier to match.
	CategoryID string
	// MinScore is the minimum affinity score threshold (inclusive).
	MinScore float64
}

// AffinityVariationFilter defines a product variation affinity segment filter constraint.
type AffinityVariationFilter struct {
	// Name is the variation attribute name (e.g. "color").
	Name string
	// Value is the variation attribute value (e.g. "black").
	Value string
	// MinScore is the minimum affinity score threshold (inclusive).
	MinScore float64
}

// SegmentFilter defines analytical segment filter values.
type SegmentFilter struct {
	// CityCodes defines optional city-code filters.
	CityCodes []string
	// MinTotalSpend defines optional minimum order total filters.
	MinTotalSpend *float64
	// RequireEmailOptIn defines optional email opt-in requirements.
	RequireEmailOptIn *bool
	// PurchasedSKUs defines optional purchased SKU filters (matched with IN — any one match is sufficient).
	PurchasedSKUs []string
	// OrderRecencyDays defines optional "ordered in last N days" filters.
	OrderRecencyDays *int
	// NoOrderRecencyDays defines optional "no orders in last N days" filters.
	NoOrderRecencyDays *int
	// CategoryPattern defines optional category or item-pattern filters.
	CategoryPattern string
	// TopSpendersPercentage defines optional top-spender percentage filters.
	TopSpendersPercentage *float64
	// TopSpendersLimit defines optional top-spender absolute-limit filters.
	TopSpendersLimit *int
	// FirstPurchaseOnly filters contacts with exactly one order.
	FirstPurchaseOnly bool
	// SubscribedNoBuy filters contacts opted in but with no orders.
	SubscribedNoBuy bool
	// OptInChannel defines optional membership-channel filters.
	OptInChannel string
	// OptInAction defines optional membership-action filters.
	OptInAction string
	// MetadataKey defines optional contact metadata-key filters.
	MetadataKey string
	// MetadataValue defines optional contact metadata-value filters.
	MetadataValue string
	// OrderStatuses defines optional order current-status inclusion filters.
	// When non-empty, only orders whose current_status is in this list are considered.
	OrderStatuses []string
	// RFMGroup defines an optional RFM group slug to filter by.
	RFMGroup string
	// RFMScoreMin defines an optional minimum RFM total score filter.
	RFMScoreMin *int
	// RFMScoreMax defines an optional maximum RFM total score filter.
	RFMScoreMax *int
	// RFMRMin defines an optional minimum R-band score filter.
	RFMRMin *int
	// RFMRMax defines an optional maximum R-band score filter.
	RFMRMax *int
	// RFMFMin defines an optional minimum F-band score filter.
	RFMFMin *int
	// RFMFMax defines an optional maximum F-band score filter.
	RFMFMax *int
	// RFMMMin defines an optional minimum M-band score filter.
	RFMMMin *int
	// RFMMMax defines an optional maximum M-band score filter.
	RFMMMax *int
	// AffinityTags defines optional tag-affinity segment filter constraints.
	AffinityTags []AffinityTagFilter
	// AffinityCategories defines optional category-affinity segment filter constraints.
	AffinityCategories []AffinityCategoryFilter
	// AffinityVariations defines optional variation-affinity segment filter constraints.
	AffinityVariations []AffinityVariationFilter
}
