// Package domain defines coupon aggregate values and domain invariants.
package domain

import (
	"errors"
	"math/rand"
	"strings"
	"time"
)

// DiscountType defines supported coupon discount types.
type DiscountType string

const (
	// DiscountTypeFixed defines fixed-amount discount values.
	DiscountTypeFixed DiscountType = "fixed"
	// DiscountTypePercentage defines percentage-based discount values.
	DiscountTypePercentage DiscountType = "percentage"
)

// codeCharset defines the character set used for random coupon code generation.
// Excludes visually ambiguous characters: 0/O, 1/I/L.
const codeCharset = "BCDFGHJKMNPQRSTVWXYZ23456789"

var (
	// ErrCodeRequired is returned when coupon codes are blank.
	ErrCodeRequired = errors.New("coupon code is required")
	// ErrCodeTooLong is returned when coupon codes exceed the maximum length.
	ErrCodeTooLong = errors.New("coupon code must not exceed 128 characters")
	// ErrDiscountTypeInvalid is returned when discount type values are unsupported.
	ErrDiscountTypeInvalid = errors.New("coupon discount type must be fixed or percentage")
	// ErrDiscountAmountInvalid is returned when discount amounts are negative or zero.
	ErrDiscountAmountInvalid = errors.New("coupon discount amount must be greater than zero")
	// ErrPercentageExceedsMax is returned when percentage discount values exceed 100.
	ErrPercentageExceedsMax = errors.New("coupon percentage discount must not exceed 100")
)

// Coupon defines a coupon aggregate with assignment, usage, and scope rules.
type Coupon struct {
	// ID defines unique coupon identifiers.
	ID string
	// Code defines unique, uppercase coupon codes.
	Code string
	// Origin defines the source that created this coupon (e.g., "manual", "campaign", "woocommerce").
	Origin string
	// DiscountType defines the discount calculation method.
	DiscountType DiscountType
	// DiscountAmount defines the discount value (currency units for fixed, percent for percentage).
	DiscountAmount float64
	// MaxUsagesGlobal defines the global maximum redemption limit. Nil means unlimited.
	MaxUsagesGlobal *int
	// MaxUsagesPerEmail defines the per-email maximum redemption limit. Nil means unlimited.
	MaxUsagesPerEmail *int
	// Active reports whether this coupon is currently active.
	Active bool
	// ExpiresAt defines the optional coupon expiry timestamp.
	ExpiresAt *time.Time
	// AssignedEmails defines the optional list of emails authorized to redeem this coupon.
	// Empty means any email may redeem it (subject to other limits).
	AssignedEmails []string
	// AssignedContactIDs defines the optional list of contact identifiers authorized to redeem this coupon.
	AssignedContactIDs []string
	// IncludedProductIDs defines products this coupon applies to. Empty means all products.
	IncludedProductIDs []string
	// IncludedCategoryIDs defines product categories this coupon applies to. Empty means all categories.
	IncludedCategoryIDs []string
	// IncludedTagIDs defines product tags this coupon applies to. Empty means all tags.
	// Note: tag filtering is enforced by our system only; WooCommerce does not natively support coupon-tag restrictions.
	IncludedTagIDs []string
	// WooCommerceID defines the optional WooCommerce coupon identifier for sync deduplication.
	WooCommerceID *int
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time
}

// Normalize canonicalizes coupon values before validation and persistence.
func (c *Coupon) Normalize() {
	if c == nil {
		return
	}

	c.Code = strings.ToUpper(strings.TrimSpace(c.Code))
	c.Origin = strings.TrimSpace(c.Origin)
	c.DiscountType = DiscountType(strings.TrimSpace(string(c.DiscountType)))
	if c.DiscountAmount < 0 {
		c.DiscountAmount = 0
	}

	for i := range c.AssignedEmails {
		c.AssignedEmails[i] = strings.ToLower(strings.TrimSpace(c.AssignedEmails[i]))
	}
	c.AssignedEmails = filterEmpty(c.AssignedEmails)

	for i := range c.AssignedContactIDs {
		c.AssignedContactIDs[i] = strings.TrimSpace(c.AssignedContactIDs[i])
	}
	c.AssignedContactIDs = filterEmpty(c.AssignedContactIDs)

	for i := range c.IncludedProductIDs {
		c.IncludedProductIDs[i] = strings.TrimSpace(c.IncludedProductIDs[i])
	}
	c.IncludedProductIDs = filterEmpty(c.IncludedProductIDs)

	for i := range c.IncludedCategoryIDs {
		c.IncludedCategoryIDs[i] = strings.TrimSpace(c.IncludedCategoryIDs[i])
	}
	c.IncludedCategoryIDs = filterEmpty(c.IncludedCategoryIDs)

	for i := range c.IncludedTagIDs {
		c.IncludedTagIDs[i] = strings.TrimSpace(c.IncludedTagIDs[i])
	}
	c.IncludedTagIDs = filterEmpty(c.IncludedTagIDs)
}

// Validate validates coupon aggregate invariants.
func (c Coupon) Validate() error {
	if strings.TrimSpace(c.Code) == "" {
		return ErrCodeRequired
	}
	if len(c.Code) > 128 {
		return ErrCodeTooLong
	}
	if err := validateDiscountType(c.DiscountType); err != nil {
		return err
	}
	if c.DiscountAmount <= 0 {
		return ErrDiscountAmountInvalid
	}
	if c.DiscountType == DiscountTypePercentage && c.DiscountAmount > 100 {
		return ErrPercentageExceedsMax
	}

	return nil
}

// GenerateCode generates a random coupon code in the format XXXX-XXXX-XXXX.
// Uses a charset that excludes visually ambiguous characters (0/O, 1/I/L).
func GenerateCode() string {
	segments := make([]string, 3)
	for s := range segments {
		buf := make([]byte, 4)
		for i := range buf {
			buf[i] = codeCharset[rand.Intn(len(codeCharset))]
		}
		segments[s] = string(buf)
	}

	return strings.Join(segments, "-")
}

// validateDiscountType validates discount type values.
func validateDiscountType(value DiscountType) error {
	switch value {
	case DiscountTypeFixed, DiscountTypePercentage:
		return nil
	default:
		return ErrDiscountTypeInvalid
	}
}

// filterEmpty removes empty strings from a slice without allocating a new backing array when unnecessary.
func filterEmpty(values []string) []string {
	if len(values) == 0 {
		return values
	}

	result := values[:0]
	for _, v := range values {
		if v != "" {
			result = append(result, v)
		}
	}
	if len(result) == 0 {
		return nil
	}

	return result
}
