package runtime

import (
	"testing"
	"time"
)

// TestIsBidirectionalSyncEnabled verifies sync mode normalization.
func TestIsBidirectionalSyncEnabled(t *testing.T) {
	if !isBidirectionalSyncEnabled(" bidirectional ") {
		t.Fatal("bidirectional mode should enable local-to-Shopify sync")
	}
	if isBidirectionalSyncEnabled("") {
		t.Fatal("empty mode should default to Shopify-to-Mannaiah only")
	}
	if isBidirectionalSyncEnabled("shopify") {
		t.Fatal("shopify mode should disable local-to-Shopify sync")
	}
}

// TestResolveDurationMS verifies millisecond duration config mapping.
func TestResolveDurationMS(t *testing.T) {
	if resolveDurationMS(1200) != 1200*time.Millisecond {
		t.Fatalf("resolveDurationMS(1200) = %s", resolveDurationMS(1200))
	}
	if resolveDurationMS(0) != 0 {
		t.Fatal("resolveDurationMS(0) should disable optional duration")
	}
}
