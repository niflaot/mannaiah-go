package domain

// TagAffinity defines tag affinity score values for one contact.
type TagAffinity struct {
	// ContactID identifies the contact.
	ContactID string
	// Tag defines the product tag value.
	Tag string
	// AffinityScore is a time-decayed purchase-weighted affinity score.
	AffinityScore float64
	// TotalSpent is the total monetary value of purchases tagged with this tag.
	TotalSpent float64
	// PurchaseCount is the total number of purchase events for this tag.
	PurchaseCount uint32
}

// CategoryAffinity defines category affinity score values for one contact.
type CategoryAffinity struct {
	// ContactID identifies the contact.
	ContactID string
	// CategoryID identifies the product category.
	CategoryID string
	// CategoryName is the human-readable category name.
	CategoryName string
	// AffinityScore is a time-decayed purchase-weighted affinity score.
	AffinityScore float64
	// TotalSpent is the total monetary value of purchases in this category.
	TotalSpent float64
	// PurchaseCount is the total number of purchase events in this category.
	PurchaseCount uint32
}

// VariationAffinity defines product variation affinity score values for one contact.
type VariationAffinity struct {
	// ContactID identifies the contact.
	ContactID string
	// VariationName is the variation attribute name (e.g. "color").
	VariationName string
	// VariationValue is the variation attribute value (e.g. "black").
	VariationValue string
	// AffinityScore is a time-decayed purchase-weighted affinity score.
	AffinityScore float64
	// TotalSpent is the total monetary value of purchases of this variation.
	TotalSpent float64
	// PurchaseCount is the total number of purchase events for this variation.
	PurchaseCount uint32
}

// AffinityProfile aggregates all affinity dimensions for one contact.
type AffinityProfile struct {
	// ContactID identifies the contact.
	ContactID string
	// Tags contains ranked tag affinity scores for the contact.
	Tags []TagAffinity
	// Categories contains ranked category affinity scores for the contact.
	Categories []CategoryAffinity
	// Variations contains ranked variation affinity scores for the contact.
	Variations []VariationAffinity
}
