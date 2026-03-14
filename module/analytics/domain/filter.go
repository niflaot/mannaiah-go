package domain

// SegmentFilter defines analytical segment filter values.
type SegmentFilter struct {
	// CityCodes defines optional city-code filters.
	CityCodes []string
	// MinTotalSpend defines optional minimum order total filters.
	MinTotalSpend *float64
	// RequireEmailOptIn defines optional email opt-in requirements.
	RequireEmailOptIn *bool
	// PurchasedSKU defines optional purchased SKU filters.
	PurchasedSKU string
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
}
