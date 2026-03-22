package runtime

import "testing"

// TestResolveTrackingBaseURL verifies explicit and sender-domain fallback behavior.
func TestResolveTrackingBaseURL(t *testing.T) {
	t.Parallel()

	if value := resolveTrackingBaseURL("https://api.flockstore.co", "contacto@flockstore.co"); value != "https://api.flockstore.co" {
		t.Fatalf("resolveTrackingBaseURL(explicit) = %q, want explicit value", value)
	}

	if value := resolveTrackingBaseURL("", "contacto@flockstore.co"); value != "https://flockstore.co" {
		t.Fatalf("resolveTrackingBaseURL(sender fallback) = %q, want sender domain fallback", value)
	}

	if value := resolveTrackingBaseURL("", ""); value != "" {
		t.Fatalf("resolveTrackingBaseURL(empty) = %q, want empty", value)
	}
}
