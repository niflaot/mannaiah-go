package runtime

import (
	"testing"
	"time"
)

// TestResolveDurationMS verifies millisecond duration config mapping.
func TestResolveDurationMS(t *testing.T) {
	if resolveDurationMS(1200) != 1200*time.Millisecond {
		t.Fatalf("resolveDurationMS(1200) = %s", resolveDurationMS(1200))
	}
	if resolveDurationMS(0) != 0 {
		t.Fatal("resolveDurationMS(0) should disable optional duration")
	}
}
