package domain

// SegmentFilter defines analytical segment filter values.
type SegmentFilter struct {
	// CityCodes defines optional city-code filters.
	CityCodes []string
	// MinTotalSpend defines optional minimum order total filters.
	MinTotalSpend *float64
	// RequireEmailOptIn defines whether email opt-in is required.
	RequireEmailOptIn bool
	// PurchasedSKU defines optional purchased SKU filters.
	PurchasedSKU string
}
