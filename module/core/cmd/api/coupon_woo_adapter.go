package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	couponservice "mannaiah/module/coupons/application/coupon/service"
	coupondomain "mannaiah/module/coupons/domain"
	woocommerceport "mannaiah/module/woocommerce/port"
)

// couponWooSyncAdapter adapts the coupons service to satisfy port.CouponSyncTarget.
type couponWooSyncAdapter struct {
	service *couponservice.Service
}

// SyncUsageByCode backfills coupon usage matched by coupon code for WooCommerce order syncs.
func (a couponWooSyncAdapter) SyncUsageByCode(ctx context.Context, cmd couponservice.SyncUsageByCodeCommand) error {
	if a.service == nil {
		return nil
	}

	return a.service.SyncUsageByCode(ctx, cmd)
}

// UpsertByWooID creates or updates a coupon keyed by its WooCommerce identifier.
func (a couponWooSyncAdapter) UpsertByWooID(ctx context.Context, coupon woocommerceport.WooCoupon) (woocommerceport.UpsertOutcome, error) {
	existing, err := a.service.GetByWooCommerceID(ctx, coupon.ID)
	if err != nil {
		return "", fmt.Errorf("lookup coupon by woocommerce id %d: %w", coupon.ID, err)
	}

	if existing == nil {
		return a.createFromWoo(ctx, coupon)
	}

	return a.updateFromWoo(ctx, *existing, coupon)
}

// createFromWoo creates a new coupon from a WooCommerce coupon payload.
func (a couponWooSyncAdapter) createFromWoo(ctx context.Context, coupon woocommerceport.WooCoupon) (woocommerceport.UpsertOutcome, error) {
	discountAmount := parseWooDiscountAmount(coupon.Amount)
	wooID := coupon.ID

	_, err := a.service.Create(ctx, couponservice.CreateCommand{
		Code:                strings.ToUpper(strings.TrimSpace(coupon.Code)),
		Origin:              "woocommerce",
		DiscountType:        mapWooDiscountType(coupon.DiscountType),
		DiscountAmount:      discountAmount,
		MaxUsagesGlobal:     resolveWooUsageLimit(coupon.UsageLimit),
		MaxUsagesPerEmail:   resolveWooUsageLimit(coupon.UsageLimitPerUser),
		Active:              true,
		AssignedEmails:      coupon.EmailRestrictions,
		IncludedProductIDs:  intSliceToStringSlice(coupon.ProductIDs),
		IncludedCategoryIDs: intSliceToStringSlice(coupon.ProductCategories),
		WooCommerceID:       &wooID,
	})
	if err != nil {
		return "", fmt.Errorf("create coupon from woocommerce %d: %w", coupon.ID, err)
	}

	return woocommerceport.UpsertOutcomeCreated, nil
}

// updateFromWoo updates an existing coupon from a WooCommerce coupon payload.
func (a couponWooSyncAdapter) updateFromWoo(ctx context.Context, existing coupondomain.Coupon, coupon woocommerceport.WooCoupon) (woocommerceport.UpsertOutcome, error) {
	discountAmount := parseWooDiscountAmount(coupon.Amount)
	discountType := mapWooDiscountType(coupon.DiscountType)

	if !wooFieldsChanged(existing, coupon, discountType, discountAmount) {
		return woocommerceport.UpsertOutcomeUnchanged, nil
	}

	_, err := a.service.Update(ctx, couponservice.UpdateCommand{
		ID:                  existing.ID,
		Origin:              "woocommerce",
		DiscountType:        discountType,
		DiscountAmount:      discountAmount,
		MaxUsagesGlobal:     resolveWooUsageLimit(coupon.UsageLimit),
		MaxUsagesPerEmail:   resolveWooUsageLimit(coupon.UsageLimitPerUser),
		Active:              existing.Active,
		ExpiresAt:           existing.ExpiresAt,
		AssignedEmails:      coupon.EmailRestrictions,
		IncludedProductIDs:  intSliceToStringSlice(coupon.ProductIDs),
		IncludedCategoryIDs: intSliceToStringSlice(coupon.ProductCategories),
		IncludedTagIDs:      existing.IncludedTagIDs,
	})
	if err != nil {
		return "", fmt.Errorf("update coupon from woocommerce %d: %w", coupon.ID, err)
	}

	return woocommerceport.UpsertOutcomeUpdated, nil
}

// wooFieldsChanged reports whether any WooCommerce-owned fields differ from the existing coupon.
func wooFieldsChanged(existing coupondomain.Coupon, coupon woocommerceport.WooCoupon, discountType coupondomain.DiscountType, discountAmount float64) bool {
	if existing.DiscountType != discountType {
		return true
	}
	if existing.DiscountAmount != discountAmount {
		return true
	}
	existingGlobal := 0
	if existing.MaxUsagesGlobal != nil {
		existingGlobal = *existing.MaxUsagesGlobal
	}
	if existingGlobal != coupon.UsageLimit {
		return true
	}
	existingPerEmail := 0
	if existing.MaxUsagesPerEmail != nil {
		existingPerEmail = *existing.MaxUsagesPerEmail
	}
	if existingPerEmail != coupon.UsageLimitPerUser {
		return true
	}

	return false
}

// mapWooDiscountType maps WooCommerce discount type strings to domain discount types.
func mapWooDiscountType(wooType string) coupondomain.DiscountType {
	switch strings.TrimSpace(wooType) {
	case "percent":
		return coupondomain.DiscountTypePercentage
	default:
		return coupondomain.DiscountTypeFixed
	}
}

// parseWooDiscountAmount parses WooCommerce discount amount strings to float64.
func parseWooDiscountAmount(amount string) float64 {
	value, _ := strconv.ParseFloat(strings.TrimSpace(amount), 64)
	return value
}

// resolveWooUsageLimit maps WooCommerce usage limit integers to optional limit pointers.
func resolveWooUsageLimit(limit int) *int {
	if limit <= 0 {
		return nil
	}
	resolved := limit
	return &resolved
}

// intSliceToStringSlice converts integer slices to string slices.
func intSliceToStringSlice(values []int) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, v := range values {
		result = append(result, strconv.Itoa(v))
	}
	return result
}
