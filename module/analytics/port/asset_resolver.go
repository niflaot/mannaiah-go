package port

import "context"

// AssetURLResolver defines asset URL resolution behavior for recommendation display data.
type AssetURLResolver interface {
	// ResolveURL returns the public URL for an asset identifier.
	// Returns empty string if the asset cannot be resolved.
	ResolveURL(ctx context.Context, assetID string) string
}

// NoopAssetURLResolver returns empty strings for all assets.
type NoopAssetURLResolver struct{}

// ResolveURL returns empty string.
func (NoopAssetURLResolver) ResolveURL(_ context.Context, _ string) string { return "" }
