package main

import (
	"context"
	"errors"
	"testing"

	assetsdomain "mannaiah/module/assets/domain"
)

// analyticsAssetLookupServiceMock defines asset lookup behavior for resolver tests.
type analyticsAssetLookupServiceMock struct {
	// assets defines asset values returned by id.
	assets map[string]*assetsdomain.Asset
	// err defines optional lookup failure values.
	err error
}

// Get resolves configured asset values.
func (m analyticsAssetLookupServiceMock) Get(_ context.Context, id string) (*assetsdomain.Asset, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.assets[id], nil
}

// TestAnalyticsAssetURLResolverResolveURLFromMetadata verifies metadata URL priority behavior.
func TestAnalyticsAssetURLResolverResolveURLFromMetadata(t *testing.T) {
	t.Parallel()

	resolver := analyticsAssetURLResolver{
		assetService: analyticsAssetLookupServiceMock{
			assets: map[string]*assetsdomain.Asset{
				"asset-1": {
					ID: "asset-1",
					Metadata: map[string]string{
						"url":        "https://cdn.example.com/fallback.jpg",
						"public_url": "https://cdn.example.com/preferred.jpg",
					},
				},
			},
		},
		assetBaseURL: "https://storage.example.com/fl-assets",
	}

	value := resolver.ResolveURL(context.Background(), "asset-1")
	if value != "https://cdn.example.com/preferred.jpg" {
		t.Fatalf("ResolveURL() = %q, want metadata preferred url", value)
	}
}

// TestAnalyticsAssetURLResolverResolveURLFromAssetKey verifies public URL fallback behavior from asset keys.
func TestAnalyticsAssetURLResolverResolveURLFromAssetKey(t *testing.T) {
	t.Parallel()

	resolver := analyticsAssetURLResolver{
		assetService: analyticsAssetLookupServiceMock{
			assets: map[string]*assetsdomain.Asset{
				"asset-1": {
					ID:       "asset-1",
					Key:      "assets/catalog/product-1.jpg",
					Metadata: map[string]string{},
				},
			},
		},
		assetBaseURL: "https://storageapi.flockstore.co/fl-assets/",
	}

	value := resolver.ResolveURL(context.Background(), "asset-1")
	if value != "https://storageapi.flockstore.co/fl-assets/assets/catalog/product-1.jpg" {
		t.Fatalf("ResolveURL() = %q, want key-based public url", value)
	}
}

// TestAnalyticsAssetURLResolverResolveURLHandlesLookupError verifies graceful resolver behavior on lookup errors.
func TestAnalyticsAssetURLResolverResolveURLHandlesLookupError(t *testing.T) {
	t.Parallel()

	resolver := analyticsAssetURLResolver{
		assetService: analyticsAssetLookupServiceMock{
			err: errors.New("lookup failed"),
		},
		assetBaseURL: "https://storage.example.com/fl-assets",
	}

	value := resolver.ResolveURL(context.Background(), "asset-1")
	if value != "" {
		t.Fatalf("ResolveURL() = %q, want empty string", value)
	}
}

// TestResolveMarketingAssetBaseURL verifies base URL candidate selection behavior.
func TestResolveMarketingAssetBaseURL(t *testing.T) {
	t.Parallel()

	fallback := buildStorageBucketBaseURL("https://storageapi.flockstore.co/", "/fl-assets/")
	resolved := resolveMarketingAssetBaseURL("", "   ", fallback)
	if resolved != "https://storageapi.flockstore.co/fl-assets" {
		t.Fatalf("resolveMarketingAssetBaseURL() = %q, want storage bucket base url", resolved)
	}

	explicit := resolveMarketingAssetBaseURL("https://cdn.example.com/assets", fallback)
	if explicit != "https://cdn.example.com/assets" {
		t.Fatalf("resolveMarketingAssetBaseURL(explicit) = %q, want explicit base url", explicit)
	}
}
