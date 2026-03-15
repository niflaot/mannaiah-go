package clickhouse

import (
	"strings"
	"testing"

	"mannaiah/module/analytics/domain"
)

// TestBuildSegmentWhereIncludesExtendedFilters verifies SQL and arg expansion for extended filters.
func TestBuildSegmentWhereIncludesExtendedFilters(t *testing.T) {
	requireOptIn := true
	minSpend := 120.5
	recency := 30
	noRecency := 90
	percentage := 10.0
	filter := domain.SegmentFilter{
		CityCodes:             []string{"BOG", "MDE"},
		RequireEmailOptIn:     &requireOptIn,
		MinTotalSpend:         &minSpend,
		PurchasedSKU:          "SKU-1",
		CategoryPattern:       "snack",
		OrderRecencyDays:      &recency,
		NoOrderRecencyDays:    &noRecency,
		TopSpendersPercentage: &percentage,
		FirstPurchaseOnly:     true,
		SubscribedNoBuy:       true,
		OptInChannel:          "sms",
		OptInAction:           "opt_in",
		MetadataKey:           "segment.group",
		MetadataValue:         "vip",
	}

	whereSQL, args := buildSegmentWhere(filter, []string{"c-1", "c-2"})
	for _, fragment := range []string{
		"cs.city_code IN (?,?)",
		"membership_events",
		"orders_fact of FINAL",
		"order_items_fact oi FINAL",
		"JSONExtractString",
		"cs.contact_id IN (?,?)",
	} {
		if !strings.Contains(whereSQL, fragment) {
			t.Fatalf("whereSQL missing fragment %q", fragment)
		}
	}
	if len(args) == 0 {
		t.Fatalf("args should not be empty")
	}
}

// TestResolveTopSpenderLimit verifies top-spender limit resolution behavior.
func TestResolveTopSpenderLimit(t *testing.T) {
	limit := 25
	percentage := 10.0

	resolved := resolveTopSpenderLimit(domain.SegmentFilter{TopSpendersLimit: &limit, TopSpendersPercentage: &percentage}, 100)
	if resolved != 25 {
		t.Fatalf("resolveTopSpenderLimit(limit) = %d, want 25", resolved)
	}

	resolved = resolveTopSpenderLimit(domain.SegmentFilter{TopSpendersPercentage: &percentage}, 123)
	if resolved != 13 {
		t.Fatalf("resolveTopSpenderLimit(percentage) = %d, want 13", resolved)
	}
}
