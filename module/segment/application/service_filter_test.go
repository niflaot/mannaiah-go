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
	if len(mapped.Clauses) != len(filters) {
		t.Fatalf("len(mapped.Clauses) = %d, want %d", len(mapped.Clauses), len(filters))
	}
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

// TestToAnalyticsFilterMapsExcludeClauses verifies exclude clauses are preserved in analytics mapping.
func TestToAnalyticsFilterMapsExcludeClauses(t *testing.T) {
	mapped := toAnalyticsFilter([]domain.Filter{
		{Type: "order_recency", Parameters: map[string]any{"days": float64(90)}},
		{Type: "order_recency", Exclude: true, Parameters: map[string]any{"days": float64(30)}},
		{Type: "city", Exclude: true, Parameters: map[string]any{"codes": []any{"BOG"}}},
	})

	if len(mapped.Clauses) != 3 {
		t.Fatalf("len(mapped.Clauses) = %d, want 3", len(mapped.Clauses))
	}
	if mapped.Clauses[1].Type != "order_recency" || !mapped.Clauses[1].Exclude {
		t.Fatalf("mapped.Clauses[1] = %#v, want excluded order_recency clause", mapped.Clauses[1])
	}
	if mapped.Clauses[2].Type != "city" || !mapped.Clauses[2].Exclude {
		t.Fatalf("mapped.Clauses[2] = %#v, want excluded city clause", mapped.Clauses[2])
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

	err = validateFilters([]domain.Filter{{Type: "subscribed_no_buy"}})
	if err != nil {
		t.Fatalf("validateFilters(subscribed_no_buy) error = %v, want nil", err)
	}

	err = validateFilters([]domain.Filter{{Type: "rfm_range", Parameters: map[string]any{"rMin": float64(3)}}})
	if err != nil {
		t.Fatalf("validateFilters(rfm_range with rMin) error = %v, want nil", err)
	}
}

// TestAffinityFiltersRequirePercentage verifies affinity filters accept percentage thresholds only.
func TestAffinityFiltersRequirePercentage(t *testing.T) {
	valid := []domain.Filter{
		{
			Type: "tag_affinity",
			Parameters: map[string]any{"tags": []any{
				map[string]any{"tag": "gimnasio", "minScorePct": float64(70), "relatedTags": []any{"deportivo", "urbano"}},
			}},
		},
		{
			Type: "category_affinity",
			Parameters: map[string]any{"categories": []any{
				map[string]any{"categoryId": "c-1", "minScorePct": float64(65)},
			}},
		},
		{
			Type: "variation_affinity",
			Parameters: map[string]any{"variations": []any{
				map[string]any{"name": "size", "value": "grande", "minScorePct": float64(55)},
			}},
		},
	}
	if err := validateFilters(valid); err != nil {
		t.Fatalf("validateFilters(valid affinity pct filters) error = %v, want nil", err)
	}

	legacy := []domain.Filter{
		{
			Type: "tag_affinity",
			Parameters: map[string]any{"tags": []any{
				map[string]any{"tag": "gimnasio", "minScore": float64(126056.94)},
			}},
		},
	}
	if err := validateFilters(legacy); err == nil {
		t.Fatalf("validateFilters(legacy minScore affinity filter) expected error")
	}
}

// TestToAnalyticsFilterMapsAffinityPercentage verifies affinity percentage and related-tag mapping.
func TestToAnalyticsFilterMapsAffinityPercentage(t *testing.T) {
	mapped := toAnalyticsFilter([]domain.Filter{
		{
			Type: "tag_affinity",
			Parameters: map[string]any{"tags": []any{
				map[string]any{"tag": "gimnasio", "relatedTags": []any{"deportivo", "urbano"}, "minScorePct": float64(70)},
			}},
		},
	})

	if len(mapped.AffinityTags) != 1 {
		t.Fatalf("len(mapped.AffinityTags) = %d, want 1", len(mapped.AffinityTags))
	}
	row := mapped.AffinityTags[0]
	if row.Tag != "gimnasio" {
		t.Fatalf("mapped.AffinityTags[0].Tag = %q, want gimnasio", row.Tag)
	}
	if row.MinScorePct != 70 {
		t.Fatalf("mapped.AffinityTags[0].MinScorePct = %v, want 70", row.MinScorePct)
	}
	if len(row.RelatedTags) != 2 {
		t.Fatalf("len(mapped.AffinityTags[0].RelatedTags) = %d, want 2", len(row.RelatedTags))
	}
}

// TestValidateMailOpenRateFilter verifies mail_open_rate filter validation behavior.
func TestValidateMailOpenRateFilter(t *testing.T) {
	err := validateFilters([]domain.Filter{
		{Type: "mail_open_rate", Parameters: map[string]any{"min": float64(50)}},
	})
	if err != nil {
		t.Fatalf("validateFilters(mail_open_rate min=50) error = %v, want nil", err)
	}

	err = validateFilters([]domain.Filter{
		{Type: "mail_open_rate", Parameters: map[string]any{"max": float64(80)}},
	})
	if err != nil {
		t.Fatalf("validateFilters(mail_open_rate max=80) error = %v, want nil", err)
	}

	err = validateFilters([]domain.Filter{
		{Type: "mail_open_rate", Parameters: map[string]any{"min": float64(30), "max": float64(90)}},
	})
	if err != nil {
		t.Fatalf("validateFilters(mail_open_rate min+max) error = %v, want nil", err)
	}

	err = validateFilters([]domain.Filter{
		{Type: "mail_open_rate", Parameters: map[string]any{}},
	})
	if err == nil {
		t.Fatalf("validateFilters(mail_open_rate no params) expected error")
	}

	err = validateFilters([]domain.Filter{
		{Type: "mail_open_rate", Parameters: map[string]any{"min": float64(150)}},
	})
	if err == nil {
		t.Fatalf("validateFilters(mail_open_rate min=150) expected error (out of range)")
	}
}

// TestToAnalyticsFilterMapsMailOpenRate verifies mail_open_rate analytics mapping behavior.
func TestToAnalyticsFilterMapsMailOpenRate(t *testing.T) {
	mapped := toAnalyticsFilter([]domain.Filter{
		{Type: "mail_open_rate", Parameters: map[string]any{"min": float64(40), "max": float64(90)}},
	})
	if mapped.MailOpenRateMin == nil || *mapped.MailOpenRateMin != 40 {
		t.Fatalf("mapped.MailOpenRateMin = %#v, want 40", mapped.MailOpenRateMin)
	}
	if mapped.MailOpenRateMax == nil || *mapped.MailOpenRateMax != 90 {
		t.Fatalf("mapped.MailOpenRateMax = %#v, want 90", mapped.MailOpenRateMax)
	}

	mappedMinOnly := toAnalyticsFilter([]domain.Filter{
		{Type: "mail_open_rate", Parameters: map[string]any{"min": float64(25)}},
	})
	if mappedMinOnly.MailOpenRateMin == nil || *mappedMinOnly.MailOpenRateMin != 25 {
		t.Fatalf("mappedMinOnly.MailOpenRateMin = %#v, want 25", mappedMinOnly.MailOpenRateMin)
	}
	if mappedMinOnly.MailOpenRateMax != nil {
		t.Fatalf("mappedMinOnly.MailOpenRateMax = %#v, want nil", mappedMinOnly.MailOpenRateMax)
	}
}
