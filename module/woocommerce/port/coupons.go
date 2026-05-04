package port

import (
	"context"
	"time"
)

// WooCoupon defines coupon data retrieved from WooCommerce.
type WooCoupon struct {
	// ID defines WooCommerce coupon identifiers.
	ID int
	// Code defines the coupon code.
	Code string
	// DiscountType defines the WooCommerce discount type ("percent", "fixed_cart", "fixed_product").
	DiscountType string
	// Amount defines the discount amount as a string (WooCommerce returns decimals as strings).
	Amount string
	// UsageLimit defines the global usage limit (0 = unlimited).
	UsageLimit int
	// UsageLimitPerUser defines the per-user usage limit (0 = unlimited).
	UsageLimitPerUser int
	// UsageCount defines the current redemption count.
	UsageCount int
	// ProductIDs defines restricted product WooCommerce IDs.
	ProductIDs []int
	// ProductCategories defines restricted category WooCommerce IDs.
	ProductCategories []int
	// EmailRestrictions defines restricted email values.
	EmailRestrictions []string
	// MetaData defines coupon metadata values keyed by key.
	MetaData map[string]string
	// DateCreated defines WooCommerce coupon creation timestamps.
	DateCreated time.Time
	// DateModified defines WooCommerce coupon modification timestamps.
	DateModified time.Time
}

// CouponSyncCommand defines coupon upsert payload values for WooCommerce sync.
type CouponSyncCommand struct {
	// Code defines the coupon code.
	Code string
	// Origin defines the coupon origin.
	Origin string
	// DiscountType defines our discount type ("fixed" or "percentage").
	DiscountType string
	// DiscountAmount defines the discount value.
	DiscountAmount float64
	// MaxUsagesGlobal defines the global usage limit (nil = unlimited).
	MaxUsagesGlobal *int
	// MaxUsagesPerEmail defines the per-email usage limit (nil = unlimited).
	MaxUsagesPerEmail *int
	// AssignedEmails defines the list of authorized emails for WooCommerce email_restrictions.
	AssignedEmails []string
	// IncludedProductWooIDs defines WooCommerce product IDs to restrict this coupon to.
	IncludedProductWooIDs []int
	// IncludedCategoryWooIDs defines WooCommerce category IDs to restrict this coupon to.
	IncludedCategoryWooIDs []int
	// WooCommerceID defines an optional existing WooCommerce coupon ID for updates.
	WooCommerceID *int
}

// CouponSyncResult defines the result of a coupon push to WooCommerce.
type CouponSyncResult struct {
	// WooCommerceID defines the resulting WooCommerce coupon identifier.
	WooCommerceID int
	// Created reports whether the coupon was newly created (false = updated).
	Created bool
}

// CouponSource defines WooCommerce coupon retrieval behavior.
type CouponSource interface {
	// Validate verifies source connectivity and credentials.
	Validate(ctx context.Context) error
	// ListCoupons retrieves paginated coupon values and reports whether additional pages exist.
	ListCoupons(ctx context.Context, page int, pageSize int) (coupons []WooCoupon, hasNext bool, err error)
	// GetCouponByID retrieves one WooCommerce coupon by identifier.
	GetCouponByID(ctx context.Context, id int) (coupon WooCoupon, err error)
}

// CouponDestination defines WooCommerce coupon write behavior.
type CouponDestination interface {
	// UpsertCoupon creates or updates a WooCommerce coupon from a sync command.
	UpsertCoupon(ctx context.Context, command CouponSyncCommand) (CouponSyncResult, error)
}

// CouponSyncTarget defines the inbound (WooCommerce → our system) coupon upsert behavior.
type CouponSyncTarget interface {
	// UpsertByWooID creates or updates a coupon keyed by WooCommerce ID.
	UpsertByWooID(ctx context.Context, coupon WooCoupon) (UpsertOutcome, error)
}
