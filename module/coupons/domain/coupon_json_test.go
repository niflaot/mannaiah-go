package domain_test

import (
	"encoding/json"
	"testing"
	"time"

	"mannaiah/module/coupons/domain"
)

// TestCouponJSONTags verifies coupon HTTP payload field names match the documented camelCase contract.
func TestCouponJSONTags(t *testing.T) {
	usedAt := time.Date(2026, time.April, 13, 12, 0, 0, 0, time.UTC)
	wooID := 1042

	payload, err := json.Marshal(domain.Coupon{
		ID:                 "coupon-1",
		Code:               "WELCOME10",
		Origin:             "woocommerce",
		DiscountType:       domain.DiscountTypeFixed,
		DiscountAmount:     15000,
		AssignedEmails:     []string{"one@example.com"},
		AssignedContactIDs: []string{"contact-1"},
		IncludedProductIDs: []string{"product-1"},
		WooCommerceID:      &wooID,
		CreatedAt:          usedAt,
		UpdatedAt:          usedAt,
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	decoded := map[string]any{}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	assertKey(t, decoded, "id")
	assertKey(t, decoded, "code")
	assertKey(t, decoded, "origin")
	assertKey(t, decoded, "discountType")
	assertKey(t, decoded, "discountAmount")
	assertKey(t, decoded, "assignedEmails")
	assertKey(t, decoded, "assignedContactIds")
	assertKey(t, decoded, "includedProductIds")
	assertKey(t, decoded, "wooCommerceId")
	assertKey(t, decoded, "createdAt")
	assertKey(t, decoded, "updatedAt")

	assertMissingKey(t, decoded, "ID")
	assertMissingKey(t, decoded, "Code")
	assertMissingKey(t, decoded, "AssignedContactIDs")
	assertMissingKey(t, decoded, "WooCommerceID")
}

func assertKey(t *testing.T, payload map[string]any, key string) {
	t.Helper()
	if _, ok := payload[key]; !ok {
		t.Fatalf("payload missing %q: %+v", key, payload)
	}
}

func assertMissingKey(t *testing.T, payload map[string]any, key string) {
	t.Helper()
	if _, ok := payload[key]; ok {
		t.Fatalf("payload unexpectedly contains %q: %+v", key, payload)
	}
}
