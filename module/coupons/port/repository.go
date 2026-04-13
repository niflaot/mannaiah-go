// Package port defines coupon module interface contracts.
package port

import (
	"context"
	"time"

	"mannaiah/module/coupons/domain"
)

// ListQuery defines coupon list filter values.
type ListQuery struct {
	// Origin filters coupons by origin values.
	Origin string
	// Active filters by active state when non-nil.
	Active *bool
	// Code filters by exact code values.
	Code string
	// Offset defines pagination offset values.
	Offset int
	// Limit defines pagination page-size values.
	Limit int
}

// SearchQuery defines coupon full-text search filter values.
type SearchQuery struct {
	// Term performs a partial free-text match across code, origin, assigned emails, and assigned contacts.
	Term string
	// Email filters coupons that have the given email in their assigned-email list (partial match).
	Email string
	// ContactID filters coupons that have the given contact identifier in their assigned-contact list (partial match).
	ContactID string
	// Code filters coupons whose code contains the given value (partial match).
	Code string
	// Origin filters coupons by exact origin value.
	Origin string
	// DiscountType filters coupons by exact discount type value.
	DiscountType string
	// Offset defines pagination offset values.
	Offset int
	// Limit defines pagination page-size values.
	Limit int
}

// CouponRepository defines coupon persistence behavior.
type CouponRepository interface {
	// Create persists a new coupon aggregate.
	Create(ctx context.Context, coupon *domain.Coupon) error
	// GetByID retrieves a coupon by its unique identifier.
	GetByID(ctx context.Context, id string) (*domain.Coupon, error)
	// GetByCode retrieves a coupon by its unique code.
	GetByCode(ctx context.Context, code string) (*domain.Coupon, error)
	// GetByWooCommerceID retrieves a coupon by its WooCommerce identifier.
	GetByWooCommerceID(ctx context.Context, wooID int) (*domain.Coupon, error)
	// Update persists mutations to an existing coupon aggregate.
	Update(ctx context.Context, coupon *domain.Coupon) error
	// Delete soft-deletes a coupon by identifier.
	Delete(ctx context.Context, id string) error
	// List retrieves paginated coupons matching the provided query.
	List(ctx context.Context, query ListQuery) ([]domain.Coupon, int64, error)
	// Search retrieves paginated coupons matching the provided full-text search query.
	Search(ctx context.Context, query SearchQuery) ([]domain.Coupon, int64, error)
	// CodeExists reports whether a coupon code is already in use.
	CodeExists(ctx context.Context, code string) (bool, error)
}

// UsageRecord defines a single coupon redemption event.
type UsageRecord struct {
	// CouponID defines the redeemed coupon identifier.
	CouponID string
	// OrderID defines the order where the coupon was applied.
	OrderID string
	// Email defines the email that redeemed the coupon.
	Email string
	// UsedAt defines the redemption timestamp.
	UsedAt time.Time
}

// CouponUsageRepository defines coupon usage tracking behavior.
type CouponUsageRepository interface {
	// RecordUsage persists a coupon redemption event.
	RecordUsage(ctx context.Context, record UsageRecord) error
	// CountGlobalUsage counts all redemptions for a coupon.
	CountGlobalUsage(ctx context.Context, couponID string) (int64, error)
	// CountUsageByEmail counts redemptions for a coupon by email.
	CountUsageByEmail(ctx context.Context, couponID string, email string) (int64, error)
	// UsageExistsForOrder reports whether a coupon was already applied to a specific order.
	UsageExistsForOrder(ctx context.Context, couponID string, orderID string) (bool, error)
}
