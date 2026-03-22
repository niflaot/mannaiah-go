package recommendation

import (
	"context"
	"testing"

	"mannaiah/module/analytics/port"
)

type noopAssetResolver struct{}

// ResolveURL returns empty URL values for fallback verification.
func (noopAssetResolver) ResolveURL(_ context.Context, _ string) string { return "" }

// TestResolveRealmURL verifies realm-first URL resolution with fallback behavior.
func TestResolveRealmURL(t *testing.T) {
	t.Parallel()

	datasheets := []port.ProductDatasheetEntry{
		{Realm: "default", URL: "https://store.example.com/default"},
		{Realm: "email", URL: "https://store.example.com/email"},
	}

	if value := resolveRealmURL(datasheets, "email", nil); value != "https://store.example.com/email" {
		t.Fatalf("resolveRealmURL(email) = %q, want email realm URL", value)
	}

	if value := resolveRealmURL(datasheets, "sms", nil); value != "https://store.example.com/default" {
		t.Fatalf("resolveRealmURL(fallback) = %q, want first non-empty URL", value)
	}
}

// TestResolveRealmURLPrefersVariationScopedURL verifies variation-scoped URL precedence for matching variations.
func TestResolveRealmURLPrefersVariationScopedURL(t *testing.T) {
	t.Parallel()

	datasheets := []port.ProductDatasheetEntry{
		{
			Realm: "email",
			URL:   "https://store.example.com/email-default",
			VariationURLs: map[string]string{
				"var-red": "https://store.example.com/email-red",
			},
		},
	}

	if value := resolveRealmURL(datasheets, "email", []string{"var-red"}); value != "https://store.example.com/email-red" {
		t.Fatalf("resolveRealmURL(variation) = %q, want variation scoped URL", value)
	}
}

// TestResolveURLVariationCandidates verifies variation URL candidate ordering and filtering behavior.
func TestResolveURLVariationCandidates(t *testing.T) {
	t.Parallel()

	candidates := resolveURLVariationCandidates(
		[]string{"v-1", "v-2", "v-3"},
		[]string{"v-2", "v-x"},
		[]string{"v-3", "v-1"},
	)
	want := []string{"v-2", "v-3", "v-1"}
	if len(candidates) != len(want) {
		t.Fatalf("len(candidates) = %d, want %d (%#v)", len(candidates), len(want), candidates)
	}
	for i := range want {
		if candidates[i] != want[i] {
			t.Fatalf("candidates[%d] = %q, want %q (%#v)", i, candidates[i], want[i], candidates)
		}
	}
}

// TestResolveGalleryImageURLFallback verifies metadata URL fallback when resolver cannot build URLs.
func TestResolveGalleryImageURLFallback(t *testing.T) {
	t.Parallel()

	value := resolveGalleryImageURL(context.Background(), port.ProductGalleryEntry{
		AssetID:  "asset-1",
		AssetURL: "https://cdn.example.com/asset-1.jpg",
	}, noopAssetResolver{})
	if value != "https://cdn.example.com/asset-1.jpg" {
		t.Fatalf("resolveGalleryImageURL() = %q, want metadata URL fallback", value)
	}
}
