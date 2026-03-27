package http

import "testing"

// TestMapProductBlockRequestsEmptySlice verifies explicit empty arrays remain explicit empty slices.
func TestMapProductBlockRequestsEmptySlice(t *testing.T) {
	t.Parallel()

	mapped := mapProductBlockRequests([]productBlockRequest{})
	if mapped == nil {
		t.Fatalf("mapProductBlockRequests(empty) returned nil, want non-nil empty slice")
	}
	if len(mapped) != 0 {
		t.Fatalf("mapProductBlockRequests(empty) len = %d, want 0", len(mapped))
	}
}

// TestMapProductBlockRequestsExcludePurchasedProducts verifies boolean exclusion mapping for purchased products.
func TestMapProductBlockRequestsExcludePurchasedProducts(t *testing.T) {
	t.Parallel()

	mapped := mapProductBlockRequests([]productBlockRequest{{
		ID:                       "hero_products",
		BaseTag:                  "tier-1",
		ExcludePurchasedProducts: true,
	}})
	if len(mapped) != 1 {
		t.Fatalf("len(mapped) = %d, want 1", len(mapped))
	}
	if !mapped[0].ExcludePurchasedProducts {
		t.Fatalf("mapped[0].ExcludePurchasedProducts = false, want true")
	}
}
