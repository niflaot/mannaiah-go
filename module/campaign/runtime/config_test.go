package runtime

import "testing"

// TestResolvedMarketingOptOutSecretPrefersPrimary verifies primary env value precedence.
func TestResolvedMarketingOptOutSecretPrefersPrimary(t *testing.T) {
	t.Parallel()

	cfg := Config{
		MarketingOptOutSecret: " primary-secret ",
	}
	if got, want := cfg.ResolvedMarketingOptOutSecret(), "primary-secret"; got != want {
		t.Fatalf("ResolvedMarketingOptOutSecret() = %q, want %q", got, want)
	}
}

// TestResolvedMarketingOptOutSecretReturnsEmpty verifies empty-secret behavior.
func TestResolvedMarketingOptOutSecretReturnsEmpty(t *testing.T) {
	t.Parallel()

	cfg := Config{
		MarketingOptOutSecret: " ",
	}
	if got, want := cfg.ResolvedMarketingOptOutSecret(), ""; got != want {
		t.Fatalf("ResolvedMarketingOptOutSecret() = %q, want %q", got, want)
	}
}
