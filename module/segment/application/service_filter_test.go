package application

import (
	"testing"

	"mannaiah/module/segment/domain"
)

// TestToAnalyticsFilterMapsExtendedFilters verifies extended filter mapping behavior.
func TestToAnalyticsFilterMapsExtendedFilters(t *testing.T) {
	filters := []domain.Filter{
		{Type: "city", Parameters: map[string]any{"codes": []any{"BOG", "MDE"}}},
		{Type: "order_recency", Parameters: map[string]any{"days": float64(30)}},
		{Type: "no_order_recency", Parameters: map[string]any{"days": int64(90)}},
		{Type: "category", Parameters: map[string]any{"pattern": "snack"}},
		{Type: "top_spenders", Parameters: map[string]any{"percentage": float64(10)}},
		{Type: "first_purchase_only", Parameters: map[string]any{"enabled": true}},
		{Type: "subscribed_no_buy", Parameters: map[string]any{"enabled": true}},
		{Type: "opt_in_status", Parameters: map[string]any{"channel": "email", "status": "opt_in"}},
		{Type: "metadata", Parameters: map[string]any{"key": "segment.group", "value": "vip"}},
		{Type: "email_opt_in", Value: true},
	}

	mapped := toAnalyticsFilter(filters)
	if len(mapped.CityCodes) != 2 {
		t.Fatalf("len(mapped.CityCodes) = %d, want 2", len(mapped.CityCodes))
	}
	if mapped.OrderRecencyDays == nil || *mapped.OrderRecencyDays != 30 {
		t.Fatalf("mapped.OrderRecencyDays = %#v, want 30", mapped.OrderRecencyDays)
	}
	if mapped.NoOrderRecencyDays == nil || *mapped.NoOrderRecencyDays != 90 {
		t.Fatalf("mapped.NoOrderRecencyDays = %#v, want 90", mapped.NoOrderRecencyDays)
	}
	if mapped.CategoryPattern != "snack" {
		t.Fatalf("mapped.CategoryPattern = %q, want %q", mapped.CategoryPattern, "snack")
	}
	if mapped.TopSpendersPercentage == nil || *mapped.TopSpendersPercentage != 10 {
		t.Fatalf("mapped.TopSpendersPercentage = %#v, want 10", mapped.TopSpendersPercentage)
	}
	if !mapped.FirstPurchaseOnly {
		t.Fatalf("mapped.FirstPurchaseOnly = %t, want true", mapped.FirstPurchaseOnly)
	}
	if !mapped.SubscribedNoBuy {
		t.Fatalf("mapped.SubscribedNoBuy = %t, want true", mapped.SubscribedNoBuy)
	}
	if mapped.OptInChannel != "email" || mapped.OptInAction != "opt_in" {
		t.Fatalf("mapped opt-in filters = %q/%q, want email/opt_in", mapped.OptInChannel, mapped.OptInAction)
	}
	if mapped.MetadataKey != "segment.group" || mapped.MetadataValue != "vip" {
		t.Fatalf("mapped metadata filters = %q/%q, want segment.group/vip", mapped.MetadataKey, mapped.MetadataValue)
	}
	if mapped.RequireEmailOptIn == nil || !*mapped.RequireEmailOptIn {
		t.Fatalf("mapped.RequireEmailOptIn = %#v, want true", mapped.RequireEmailOptIn)
	}
}

// TestToAnalyticsFilterMapsPurchasedSKUs verifies purchased_sku multi-value and legacy mapping behavior.
func TestToAnalyticsFilterMapsPurchasedSKUs(t *testing.T) {
	multiMapped := toAnalyticsFilter([]domain.Filter{
		{Type: "purchased_sku", Parameters: map[string]any{"skus": []any{"SKU-A", "SKU-B"}}},
	})
	if len(multiMapped.PurchasedSKUs) != 2 {
		t.Fatalf("len(PurchasedSKUs) = %d, want 2", len(multiMapped.PurchasedSKUs))
	}

	legacyMapped := toAnalyticsFilter([]domain.Filter{
		{Type: "purchased_sku", Value: "SKU-LEGACY"},
	})
	if len(legacyMapped.PurchasedSKUs) != 1 || legacyMapped.PurchasedSKUs[0] != "SKU-LEGACY" {
		t.Fatalf("PurchasedSKUs = %v, want [SKU-LEGACY]", legacyMapped.PurchasedSKUs)
	}

	if err := validateFilters([]domain.Filter{{Type: "purchased_sku"}}); err == nil {
		t.Fatalf("validateFilters(purchased_sku with no skus) expected error")
	}
}

// TestValidateFiltersRejectsUnknown verifies unknown filter rejection behavior.
func TestValidateFiltersRejectsUnknown(t *testing.T) {
	err := validateFilters([]domain.Filter{{Type: "does_not_exist"}})
	if err == nil {
		t.Fatalf("validateFilters() expected error")
	}
}

// TestValidateFiltersRequiresParameters verifies mandatory parameter validation behavior.
func TestValidateFiltersRequiresParameters(t *testing.T) {
	err := validateFilters([]domain.Filter{{Type: "order_recency", Parameters: map[string]any{"days": float64(0)}}})
	if err == nil {
		t.Fatalf("validateFilters(order_recency days=0) expected error")
	}

	err = validateFilters([]domain.Filter{{Type: "opt_in_status", Parameters: map[string]any{"channel": "email"}}})
	if err == nil {
		t.Fatalf("validateFilters(opt_in_status without status) expected error")
	}
}
