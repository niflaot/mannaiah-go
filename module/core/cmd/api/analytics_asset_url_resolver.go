package main

import (
	"context"
	"strings"

	analyticsport "mannaiah/module/analytics/port"
	assetsdomain "mannaiah/module/assets/domain"
)

// analyticsAssetLookupService defines asset lookup behavior required by recommendation image resolution.
type analyticsAssetLookupService interface {
	// Get retrieves one asset by identifier.
	Get(ctx context.Context, id string) (*assetsdomain.Asset, error)
}

// analyticsAssetURLResolver resolves recommendation image URLs from assets metadata and storage keys.
type analyticsAssetURLResolver struct {
	// assetService defines asset lookup dependencies.
	assetService analyticsAssetLookupService
	// assetBaseURL defines public base URL values used to expose storage keys.
	assetBaseURL string
}

var _ analyticsport.AssetURLResolver = (*analyticsAssetURLResolver)(nil)

// ResolveURL resolves one public URL for the provided asset identifier.
func (r analyticsAssetURLResolver) ResolveURL(ctx context.Context, assetID string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(assetID)), "http://") ||
		strings.HasPrefix(strings.ToLower(strings.TrimSpace(assetID)), "https://") {
		return strings.TrimSpace(assetID)
	}
	if r.assetService == nil {
		return ""
	}

	trimmedAssetID := strings.TrimSpace(assetID)
	if trimmedAssetID == "" {
		return ""
	}

	asset, err := r.assetService.Get(ctx, trimmedAssetID)
	if err != nil || asset == nil {
		return ""
	}
	if resolvedFromMetadata := resolvePublicAssetMetadataURL(asset.Metadata); resolvedFromMetadata != "" {
		return resolvedFromMetadata
	}

	return buildPublicAssetURL(r.assetBaseURL, asset.Key)
}

// resolveMarketingAssetBaseURL resolves the first non-empty public base URL candidate.
func resolveMarketingAssetBaseURL(candidates ...string) string {
	for _, candidate := range candidates {
		trimmed := strings.TrimSpace(candidate)
		if trimmed != "" {
			return strings.TrimRight(trimmed, "/")
		}
	}

	return ""
}

// buildStorageBucketBaseURL builds a public bucket base URL from storage endpoint and bucket values.
func buildStorageBucketBaseURL(endpoint string, bucketName string) string {
	trimmedEndpoint := strings.TrimRight(strings.TrimSpace(endpoint), "/")
	trimmedBucket := strings.Trim(strings.TrimSpace(bucketName), "/")
	if trimmedEndpoint == "" || trimmedBucket == "" {
		return ""
	}

	return trimmedEndpoint + "/" + trimmedBucket
}

// resolvePublicAssetMetadataURL resolves a public URL from known asset metadata keys.
func resolvePublicAssetMetadataURL(metadata map[string]string) string {
	if len(metadata) == 0 {
		return ""
	}

	bestURL := ""
	bestPriority := 0
	for key, rawValue := range metadata {
		priority := assetMetadataURLPriority(key)
		if priority == 0 {
			continue
		}
		value := strings.TrimSpace(rawValue)
		if value == "" {
			continue
		}
		if bestPriority == 0 || priority < bestPriority {
			bestPriority = priority
			bestURL = value
		}
	}

	return bestURL
}

// assetMetadataURLPriority resolves known public-url metadata key priorities.
func assetMetadataURLPriority(key string) int {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "public_url", "publicurl":
		return 1
	case "cdn_url", "cdnurl":
		return 2
	case "image_url", "imageurl":
		return 3
	case "url":
		return 4
	case "source_url", "origin_url", "original_url":
		return 5
	case "woocommerce_url", "woo_url":
		return 6
	case "falabella_url":
		return 7
	default:
		return 0
	}
}

// buildPublicAssetURL builds a public URL from base URL and storage key values.
func buildPublicAssetURL(baseURL string, key string) string {
	trimmedBaseURL := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	trimmedKey := strings.TrimLeft(strings.TrimSpace(key), "/")
	if trimmedBaseURL == "" || trimmedKey == "" {
		return ""
	}

	return trimmedBaseURL + "/" + trimmedKey
}
