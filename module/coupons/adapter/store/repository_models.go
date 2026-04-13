package store

import "time"

// couponRecord defines coupon root persistence schema.
type couponRecord struct {
	// ID defines coupon identifiers.
	ID string `gorm:"primaryKey;size:64"`
	// Code defines unique coupon code values.
	Code string `gorm:"size:128;not null;uniqueIndex"`
	// Origin defines coupon origin values.
	Origin string `gorm:"size:128;not null;default:''"`
	// DiscountType defines discount type values.
	DiscountType string `gorm:"size:32;not null"`
	// DiscountAmount defines discount amount values.
	DiscountAmount float64 `gorm:"type:decimal(18,4);not null;default:0"`
	// MaxUsagesGlobal defines optional global usage limit values.
	MaxUsagesGlobal *int `gorm:"default:null"`
	// MaxUsagesPerEmail defines optional per-email usage limit values.
	MaxUsagesPerEmail *int `gorm:"default:null"`
	// Active defines whether the coupon is active.
	Active bool `gorm:"not null;default:true"`
	// ExpiresAt defines optional expiry timestamp values.
	ExpiresAt *time.Time `gorm:"default:null"`
	// WooCommerceID defines optional WooCommerce coupon identifier values.
	WooCommerceID *int `gorm:"default:null;index"`
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time
	// DeletedAt defines soft-delete timestamps.
	DeletedAt *time.Time `gorm:"index"`
}

// couponAssignedEmailRecord defines coupon-assigned-email child rows.
type couponAssignedEmailRecord struct {
	// ID defines surrogate identifiers.
	ID uint `gorm:"primaryKey"`
	// CouponID defines owning coupon identifiers.
	CouponID string `gorm:"size:64;not null;index;uniqueIndex:idx_coupon_assigned_emails_coupon_email,priority:1"`
	// Email defines assigned email values.
	Email string `gorm:"size:320;not null;uniqueIndex:idx_coupon_assigned_emails_coupon_email,priority:2"`
}

// couponAssignedContactRecord defines coupon-assigned-contact child rows.
type couponAssignedContactRecord struct {
	// ID defines surrogate identifiers.
	ID uint `gorm:"primaryKey"`
	// CouponID defines owning coupon identifiers.
	CouponID string `gorm:"size:64;not null;index;uniqueIndex:idx_coupon_assigned_contacts_coupon_contact,priority:1"`
	// ContactID defines assigned contact identifier values.
	ContactID string `gorm:"size:64;not null;uniqueIndex:idx_coupon_assigned_contacts_coupon_contact,priority:2"`
}

// couponIncludedProductRecord defines coupon-included-product child rows.
type couponIncludedProductRecord struct {
	// ID defines surrogate identifiers.
	ID uint `gorm:"primaryKey"`
	// CouponID defines owning coupon identifiers.
	CouponID string `gorm:"size:64;not null;index;uniqueIndex:idx_coupon_included_products_coupon_product,priority:1"`
	// ProductID defines included product identifier values.
	ProductID string `gorm:"size:64;not null;uniqueIndex:idx_coupon_included_products_coupon_product,priority:2"`
}

// couponIncludedCategoryRecord defines coupon-included-category child rows.
type couponIncludedCategoryRecord struct {
	// ID defines surrogate identifiers.
	ID uint `gorm:"primaryKey"`
	// CouponID defines owning coupon identifiers.
	CouponID string `gorm:"size:64;not null;index;uniqueIndex:idx_coupon_included_categories_coupon_category,priority:1"`
	// CategoryID defines included category identifier values.
	CategoryID string `gorm:"size:64;not null;uniqueIndex:idx_coupon_included_categories_coupon_category,priority:2"`
}

// couponIncludedTagRecord defines coupon-included-tag child rows.
type couponIncludedTagRecord struct {
	// ID defines surrogate identifiers.
	ID uint `gorm:"primaryKey"`
	// CouponID defines owning coupon identifiers.
	CouponID string `gorm:"size:64;not null;index;uniqueIndex:idx_coupon_included_tags_coupon_tag,priority:1"`
	// TagID defines included tag identifier values.
	TagID string `gorm:"size:64;not null;uniqueIndex:idx_coupon_included_tags_coupon_tag,priority:2"`
}

// couponUsageRecord defines coupon-usage persistence rows.
type couponUsageRecord struct {
	// ID defines surrogate identifiers.
	ID uint `gorm:"primaryKey"`
	// CouponID defines redeemed coupon identifiers.
	CouponID string `gorm:"size:64;not null;index;uniqueIndex:idx_coupon_usages_coupon_order,priority:1"`
	// OrderID defines the order where the coupon was applied.
	OrderID string `gorm:"size:64;not null;uniqueIndex:idx_coupon_usages_coupon_order,priority:2"`
	// Email defines the email that redeemed the coupon.
	Email string `gorm:"size:320;not null;default:'';index:idx_coupon_usages_email,priority:2"`
	// UsedAt defines the redemption timestamp.
	UsedAt time.Time `gorm:"not null"`
}

// TableName defines storage table names.
func (couponRecord) TableName() string { return "coupons" }

// TableName defines storage table names.
func (couponAssignedEmailRecord) TableName() string { return "coupon_assigned_emails" }

// TableName defines storage table names.
func (couponAssignedContactRecord) TableName() string { return "coupon_assigned_contact_ids" }

// TableName defines storage table names.
func (couponIncludedProductRecord) TableName() string { return "coupon_included_product_ids" }

// TableName defines storage table names.
func (couponIncludedCategoryRecord) TableName() string { return "coupon_included_category_ids" }

// TableName defines storage table names.
func (couponIncludedTagRecord) TableName() string { return "coupon_included_tag_ids" }

// TableName defines storage table names.
func (couponUsageRecord) TableName() string { return "coupon_usages" }
