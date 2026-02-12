package product

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/sync/errgroup"
	productdomain "mannaiah/module/products/domain/product"
)

const (
	// assetLookupConcurrency defines maximum concurrent asset existence lookups.
	assetLookupConcurrency = 8
)

// CopyDatasheets creates a shallow copy of datasheet slices.
func CopyDatasheets(values []productdomain.Datasheet) []productdomain.Datasheet {
	if len(values) == 0 {
		return nil
	}

	copied := make([]productdomain.Datasheet, len(values))
	copy(copied, values)
	return copied
}

// validateGalleryAssets validates that all gallery asset references exist.
func validateGalleryAssets(ctx context.Context, lookup AssetLookup, gallery []productdomain.GalleryItem) error {
	if len(gallery) == 0 {
		return nil
	}

	uniqueIDs := uniqueGalleryAssetIDs(gallery)
	if len(uniqueIDs) == 0 {
		return nil
	}

	group, groupCtx := errgroup.WithContext(ctx)
	guard := make(chan struct{}, assetLookupConcurrency)

	for _, id := range uniqueIDs {
		assetID := id
		group.Go(func() error {
			select {
			case guard <- struct{}{}:
			case <-groupCtx.Done():
				return groupCtx.Err()
			}
			defer func() { <-guard }()

			exists, err := lookup.Exists(groupCtx, assetID)
			if err != nil {
				return fmt.Errorf("check product gallery asset %q: %w", assetID, err)
			}
			if !exists {
				return fmt.Errorf("%w: assetId=%s", ErrAssetNotFound, assetID)
			}

			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return err
	}

	return nil
}

// uniqueGalleryAssetIDs deduplicates gallery asset ids while preserving order.
func uniqueGalleryAssetIDs(gallery []productdomain.GalleryItem) []string {
	seen := make(map[string]struct{}, len(gallery))
	result := make([]string, 0, len(gallery))

	for _, item := range gallery {
		assetID := strings.TrimSpace(item.AssetID)
		if assetID == "" {
			continue
		}
		if _, exists := seen[assetID]; exists {
			continue
		}

		seen[assetID] = struct{}{}
		result = append(result, assetID)
	}

	return result
}
