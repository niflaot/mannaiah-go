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
		PurchasedSKUs:         []string{"SKU-1", "SKU-2"},
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
		"oi.sku IN (?,?)",
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

// TestBuildSegmentWhereFromClausesSupportsExclude verifies include/exclude clause SQL generation.
func TestBuildSegmentWhereFromClausesSupportsExclude(t *testing.T) {
	filter := domain.SegmentFilter{
		Clauses: []domain.SegmentClause{
			{Type: "order_recency", Parameters: map[string]any{"days": float64(90)}},
			{Type: "order_recency", Exclude: true, Parameters: map[string]any{"days": float64(30)}},
			{Type: "city", Exclude: true, Parameters: map[string]any{"codes": []any{"BOG"}}},
			{Type: "purchased_sku", Exclude: true, Parameters: map[string]any{"skus": []any{"SKU-1", "SKU-2"}}},
			{Type: "order_status", Parameters: map[string]any{"statuses": []any{"completed"}}},
		},
	}

	whereSQL, args := buildSegmentWhere(filter, nil)
	for _, fragment := range []string{
		"NOT (EXISTS (",
		"NOT (cs.city_code IN (?))",
		"order_items_fact oi FINAL",
		"of.current_status IN (?)",
	} {
		if !strings.Contains(whereSQL, fragment) {
			t.Fatalf("whereSQL missing fragment %q", fragment)
		}
	}
	if len(args) == 0 {
		t.Fatalf("args should not be empty")
	}
}

// TestBuildSegmentWhereFromClausesExcludedOrderStatus verifies excluded status scope generation.
func TestBuildSegmentWhereFromClausesExcludedOrderStatus(t *testing.T) {
	filter := domain.SegmentFilter{
		Clauses: []domain.SegmentClause{
			{Type: "order_status", Parameters: map[string]any{"statuses": []any{"completed"}}},
			{Type: "order_status", Exclude: true, Parameters: map[string]any{"statuses": []any{"cancelled"}}},
			{Type: "order_recency", Parameters: map[string]any{"days": float64(15)}},
		},
	}

	whereSQL, _ := buildSegmentWhere(filter, nil)
	if !strings.Contains(whereSQL, "of.current_status IN (?)") {
		t.Fatalf("whereSQL missing included status fragment")
	}
	if !strings.Contains(whereSQL, "of.current_status NOT IN (?)") {
		t.Fatalf("whereSQL missing excluded status fragment")
	}
}

// TestBuildSegmentWhereFromClausesAlwaysTrueExclude verifies internal always-true exclude clauses produce NOT(true).
func TestBuildSegmentWhereFromClausesAlwaysTrueExclude(t *testing.T) {
	filter := domain.SegmentFilter{
		Clauses: []domain.SegmentClause{
			{Type: "__always_true__", Exclude: true},
		},
	}

	whereSQL, _ := buildSegmentWhere(filter, nil)
	if !strings.Contains(whereSQL, "NOT (1 = 1)") {
		t.Fatalf("whereSQL = %q, want NOT (1 = 1)", whereSQL)
	}
}

// TestBuildSegmentWhereFromClausesAffinityPct verifies percentage-based affinity and related-tag SQL generation.
func TestBuildSegmentWhereFromClausesAffinityPct(t *testing.T) {
	filter := domain.SegmentFilter{
		Clauses: []domain.SegmentClause{
			{
				Type: "tag_affinity",
				Parameters: map[string]any{"tags": []any{
					map[string]any{"tag": "gimnasio", "relatedTags": []any{"deportivo", "urbano"}, "minScorePct": float64(70)},
				}},
			},
			{
				Type: "category_affinity",
				Parameters: map[string]any{"categories": []any{
					map[string]any{"categoryId": "cat-1", "minScorePct": float64(65)},
				}},
			},
		},
	}

	whereSQL, args := buildSegmentWhere(filter, nil)
	for _, fragment := range []string{
		"ta.tag IN (?,?,?)",
		"ta.score * 100.0 / ta.max_score",
		"ca.category_id = ?",
		"ca.score * 100.0 / ca.max_score",
	} {
		if !strings.Contains(whereSQL, fragment) {
			t.Fatalf("whereSQL missing fragment %q", fragment)
		}
	}
	if len(args) < 6 {
		t.Fatalf("len(args) = %d, want >= 6", len(args))
	}
}
