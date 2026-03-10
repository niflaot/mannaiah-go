package products

import (
	"context"
	"errors"
	"fmt"
	"strings"

	assetsdomain "mannaiah/module/assets/domain"
	"mannaiah/module/falabella/port"
	productdomain "mannaiah/module/products/domain/product"
	variationdomain "mannaiah/module/products/domain/variation"
)

var (
	// ErrNilService is returned when product-service dependencies are nil.
	ErrNilService = errors.New("products service must not be nil")
)

// service defines product-service behavior required by this adapter.
type service interface {
	// Get retrieves products by identifier.
	Get(ctx context.Context, id string) (*productdomain.Product, error)
	// List retrieves all products.
	List(ctx context.Context) ([]productdomain.Product, error)
}

// variationService defines variation-service behavior required by this adapter.
type variationService interface {
	// Get retrieves variations by identifier.
	Get(ctx context.Context, id string) (*variationdomain.Variation, error)
}

// assetService defines asset-service behavior required by this adapter.
type assetService interface {
	// Get retrieves assets by identifier.
	Get(ctx context.Context, id string) (*assetsdomain.Asset, error)
}

// Option defines catalog-construction options.
type Option func(*Catalog)

// Catalog defines product-catalog adapters backed by module/products services.
type Catalog struct {
	// service defines product-service dependencies.
	service service
	// variationService defines optional variation-service dependencies.
	variationService variationService
	// assetService defines optional asset-service dependencies.
	assetService assetService
	// assetBaseURL defines optional base URL values used to expose asset keys publicly.
	assetBaseURL string
}

var (
	// _ ensures Catalog satisfies Falabella product-catalog ports.
	_ port.ProductCatalog = (*Catalog)(nil)
)

// WithVariationService configures optional variation-service dependencies.
func WithVariationService(service variationService) Option {
	return func(catalog *Catalog) {
		if catalog == nil {
			return
		}
		catalog.variationService = service
	}
}

// WithAssetService configures optional asset-service dependencies.
func WithAssetService(service assetService) Option {
	return func(catalog *Catalog) {
		if catalog == nil {
			return
		}
		catalog.assetService = service
	}
}

// WithAssetBaseURL configures optional public base URL values used for image URL building.
func WithAssetBaseURL(baseURL string) Option {
	return func(catalog *Catalog) {
		if catalog == nil {
			return
		}
		catalog.assetBaseURL = strings.TrimSpace(baseURL)
	}
}

// NewCatalog creates Falabella product-catalog adapters.
func NewCatalog(service service, options ...Option) (*Catalog, error) {
	if service == nil {
		return nil, ErrNilService
	}

	catalog := &Catalog{service: service}
	for _, option := range options {
		if option == nil {
			continue
		}
		option(catalog)
	}

	return catalog, nil
}

// GetProduct retrieves mapped catalog products by identifier.
func (c *Catalog) GetProduct(ctx context.Context, id string) (*port.CatalogProduct, error) {
	entity, err := c.service.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get product from products module: %w", err)
	}

	mapped, err := c.mapProduct(ctx, entity)
	if err != nil {
		return nil, err
	}

	return mapped, nil
}

// ListProducts retrieves mapped catalog products.
func (c *Catalog) ListProducts(ctx context.Context) ([]port.CatalogProduct, error) {
	entities, err := c.service.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list products from products module: %w", err)
	}

	mapped := make([]port.CatalogProduct, 0, len(entities))
	for _, entity := range entities {
		copied := entity
		item, mapErr := c.mapProduct(ctx, &copied)
		if mapErr != nil {
			return nil, mapErr
		}
		mapped = append(mapped, *item)
	}

	return mapped, nil
}

// mapProduct maps product-domain entities into Falabella catalog-product values.
func (c *Catalog) mapProduct(ctx context.Context, entity *productdomain.Product) (*port.CatalogProduct, error) {
	if entity == nil {
		return nil, nil
	}

	datasheets := make([]port.CatalogDatasheet, 0, len(entity.Datasheets))
	for _, item := range entity.Datasheets {
		attributes := make(map[string]any, len(item.Attributes))
		for key, value := range item.Attributes {
			attributes[key] = value
		}
		datasheets = append(datasheets, port.CatalogDatasheet{
			Realm:       item.Realm,
			Name:        item.Name,
			Description: item.Description,
			Attributes:  attributes,
		})
	}

	variants := make([]port.CatalogVariant, 0, len(entity.Variants))
	for _, item := range entity.Variants {
		variant := port.CatalogVariant{
			SKU:          strings.TrimSpace(item.SKU),
			VariationIDs: append([]string(nil), item.VariationIDs...),
		}
		if c != nil && c.variationService != nil {
			resolvedVariations := make([]port.CatalogVariation, 0, len(item.VariationIDs))
			for _, variationID := range item.VariationIDs {
				trimmedVariationID := strings.TrimSpace(variationID)
				if trimmedVariationID == "" {
					continue
				}
				variation, err := c.variationService.Get(ctx, trimmedVariationID)
				if err != nil {
					return nil, fmt.Errorf("get variation %q from products module: %w", trimmedVariationID, err)
				}
				if variation == nil {
					continue
				}
				resolvedVariations = append(resolvedVariations, port.CatalogVariation{
					ID:         variation.ID,
					Name:       variation.Name,
					Definition: string(variation.Definition),
					Value:      variation.Value,
				})
			}
			variant.Variations = resolvedVariations
		}
		variants = append(variants, variant)
	}

	images, err := c.resolveCatalogImages(ctx, entity.Gallery)
	if err != nil {
		return nil, err
	}

	return &port.CatalogProduct{
		ID:         entity.ID,
		SKU:        entity.SKU,
		Datasheets: datasheets,
		Variants:   variants,
		Images:     images,
	}, nil
}

// resolveCatalogImages maps product gallery items into Falabella catalog image values.
func (c *Catalog) resolveCatalogImages(ctx context.Context, gallery []productdomain.GalleryItem) ([]port.CatalogImage, error) {
	if len(gallery) == 0 {
		return nil, nil
	}

	images := make([]port.CatalogImage, 0, len(gallery))
	for _, item := range gallery {
		url, err := c.resolveAssetURL(ctx, item.AssetID)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(url) == "" {
			continue
		}

		images = append(images, port.CatalogImage{
			URL:               url,
			Position:          cloneIntPointer(item.Position),
			VariationPosition: cloneIntPointer(item.VariationPosition),
			ExcludedRealms:    append([]string(nil), item.ExcludedRealms...),
			VariationIDs:      append([]string(nil), item.VariationIDs...),
		})
	}

	return images, nil
}

// resolveAssetURL resolves public image URLs from asset metadata or configured base URL values.
func (c *Catalog) resolveAssetURL(ctx context.Context, assetID string) (string, error) {
	if c == nil || c.assetService == nil {
		return "", nil
	}

	trimmedAssetID := strings.TrimSpace(assetID)
	if trimmedAssetID == "" {
		return "", nil
	}

	asset, err := c.assetService.Get(ctx, trimmedAssetID)
	if err != nil {
		return "", fmt.Errorf("get asset %q from assets module: %w", trimmedAssetID, err)
	}
	if asset == nil {
		return "", nil
	}

	if value := resolveAssetMetadataURL(asset.Metadata); value != "" {
		return value, nil
	}

	return buildAssetURL(c.assetBaseURL, asset.Key), nil
}

// resolveAssetMetadataURL resolves a public URL from known metadata keys.
func resolveAssetMetadataURL(metadata map[string]string) string {
	if len(metadata) == 0 {
		return ""
	}

	keys := []string{"falabella_url", "public_url", "publicUrl", "cdn_url", "cdnUrl", "image_url", "url"}
	for _, key := range keys {
		if value := strings.TrimSpace(metadata[key]); value != "" {
			return value
		}
	}

	return ""
}

// buildAssetURL builds public URLs from configured base URL and storage key values.
func buildAssetURL(baseURL string, key string) string {
	trimmedBaseURL := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	trimmedKey := strings.TrimLeft(strings.TrimSpace(key), "/")
	if trimmedBaseURL == "" || trimmedKey == "" {
		return ""
	}

	return trimmedBaseURL + "/" + trimmedKey
}

// cloneIntPointer copies optional integer pointer values.
func cloneIntPointer(value *int) *int {
	if value == nil {
		return nil
	}

	resolved := *value
	return &resolved
}
