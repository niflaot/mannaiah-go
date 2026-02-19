package products

import (
	"context"
	errorspkg "errors"
	"testing"

	assetsdomain "mannaiah/module/assets/domain"
	productdomain "mannaiah/module/products/domain/product"
	variationdomain "mannaiah/module/products/domain/variation"
)

// serviceMock defines product-service behavior for adapter tests.
type serviceMock struct {
	// entity defines Get() return values.
	entity *productdomain.Product
	// entities defines List() return values.
	entities []productdomain.Product
	// err defines Get()/List() error values.
	err error
}

// variationServiceMock defines variation-service behavior for adapter tests.
type variationServiceMock struct {
	// variations defines Get() return values by variation ID.
	variations map[string]*variationdomain.Variation
	// err defines Get() error values.
	err error
}

// assetServiceMock defines asset-service behavior for adapter tests.
type assetServiceMock struct {
	// assets defines Get() return values by asset ID.
	assets map[string]*assetsdomain.Asset
	// err defines Get() error values.
	err error
}

// Get returns configured product values.
func (m serviceMock) Get(ctx context.Context, id string) (*productdomain.Product, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.entity, nil
}

// List returns configured product collections.
func (m serviceMock) List(ctx context.Context) ([]productdomain.Product, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.entities, nil
}

// Get returns configured variation values.
func (m variationServiceMock) Get(ctx context.Context, id string) (*variationdomain.Variation, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.variations[id], nil
}

// Get returns configured asset values.
func (m assetServiceMock) Get(ctx context.Context, id string) (*assetsdomain.Asset, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.assets[id], nil
}

// TestNewCatalogValidation verifies constructor validation behavior.
func TestNewCatalogValidation(t *testing.T) {
	_, err := NewCatalog(nil)
	if !errorspkg.Is(err, ErrNilService) {
		t.Fatalf("NewCatalog() error = %v, want %v", err, ErrNilService)
	}
}

// TestCatalogGetProduct verifies GetProduct mapping behavior.
func TestCatalogGetProduct(t *testing.T) {
	catalog, err := NewCatalog(serviceMock{
		entity: &productdomain.Product{
			ID:  "p-1",
			SKU: "SKU-1",
			Datasheets: []productdomain.Datasheet{
				{
					Realm:       "falabella",
					Name:        "Backpack",
					Description: "Desc",
					Attributes:  map[string]any{"brand": "GENERIC"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}

	entity, getErr := catalog.GetProduct(context.Background(), "p-1")
	if getErr != nil {
		t.Fatalf("GetProduct() error = %v", getErr)
	}
	if entity.SKU != "SKU-1" {
		t.Fatalf("entity.SKU = %q, want %q", entity.SKU, "SKU-1")
	}
	if entity.Datasheets[0].Attributes["brand"] != "GENERIC" {
		t.Fatalf("entity.Datasheets[0].Attributes[brand] = %v, want %q", entity.Datasheets[0].Attributes["brand"], "GENERIC")
	}
}

// TestCatalogListProducts verifies ListProducts mapping behavior.
func TestCatalogListProducts(t *testing.T) {
	catalog, err := NewCatalog(serviceMock{
		entities: []productdomain.Product{
			{ID: "p-1", SKU: "SKU-1"},
			{ID: "p-2", SKU: "SKU-2"},
		},
	})
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}

	entities, listErr := catalog.ListProducts(context.Background())
	if listErr != nil {
		t.Fatalf("ListProducts() error = %v", listErr)
	}
	if len(entities) != 2 {
		t.Fatalf("len(entities) = %d, want %d", len(entities), 2)
	}
}

// TestCatalogGetProductWithVariants verifies variant and variation mapping behavior.
func TestCatalogGetProductWithVariants(t *testing.T) {
	catalog, err := NewCatalog(
		serviceMock{
			entity: &productdomain.Product{
				ID:  "p-1",
				SKU: "SKU-1",
				Variants: []productdomain.Variant{
					{SKU: "SKU-RED-M", VariationIDs: []string{"v-color", "v-size"}},
				},
			},
		},
		WithVariationService(variationServiceMock{
			variations: map[string]*variationdomain.Variation{
				"v-color": {ID: "v-color", Name: "Color", Definition: variationdomain.DefinitionColor, Value: "Red"},
				"v-size":  {ID: "v-size", Name: "Size", Definition: variationdomain.DefinitionSize, Value: "M"},
			},
		}),
	)
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}

	entity, getErr := catalog.GetProduct(context.Background(), "p-1")
	if getErr != nil {
		t.Fatalf("GetProduct() error = %v", getErr)
	}
	if len(entity.Variants) != 1 {
		t.Fatalf("len(entity.Variants) = %d, want %d", len(entity.Variants), 1)
	}
	if entity.Variants[0].SKU != "SKU-RED-M" {
		t.Fatalf("entity.Variants[0].SKU = %q, want %q", entity.Variants[0].SKU, "SKU-RED-M")
	}
	if len(entity.Variants[0].Variations) != 2 {
		t.Fatalf("len(entity.Variants[0].Variations) = %d, want %d", len(entity.Variants[0].Variations), 2)
	}
	if entity.Variants[0].Variations[0].Definition != "COLOR" {
		t.Fatalf("entity.Variants[0].Variations[0].Definition = %q, want %q", entity.Variants[0].Variations[0].Definition, "COLOR")
	}
}

// TestCatalogErrorPropagation verifies service error propagation behavior.
func TestCatalogErrorPropagation(t *testing.T) {
	expectedErr := errorspkg.New("boom")
	catalog, err := NewCatalog(serviceMock{err: expectedErr})
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}

	if _, getErr := catalog.GetProduct(context.Background(), "p-1"); getErr == nil {
		t.Fatalf("GetProduct() expected error")
	}
	if _, listErr := catalog.ListProducts(context.Background()); listErr == nil {
		t.Fatalf("ListProducts() expected error")
	}
}

// TestCatalogVariationLookupError verifies variation lookup error propagation behavior.
func TestCatalogVariationLookupError(t *testing.T) {
	catalog, err := NewCatalog(
		serviceMock{
			entity: &productdomain.Product{
				ID:       "p-1",
				SKU:      "SKU-1",
				Variants: []productdomain.Variant{{SKU: "SKU-RED-M", VariationIDs: []string{"v-color"}}},
			},
		},
		WithVariationService(variationServiceMock{err: errorspkg.New("boom")}),
	)
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}

	if _, getErr := catalog.GetProduct(context.Background(), "p-1"); getErr == nil {
		t.Fatalf("GetProduct() expected variation lookup error")
	}
}

// TestCatalogGetProductWithImages verifies image URL mapping behavior.
func TestCatalogGetProductWithImages(t *testing.T) {
	catalog, err := NewCatalog(
		serviceMock{
			entity: &productdomain.Product{
				ID:  "p-1",
				SKU: "SKU-1",
				Gallery: []productdomain.GalleryItem{
					{AssetID: "asset-1"},
					{AssetID: "asset-2", VariationIDs: []string{"v-color"}, ExcludedRealms: []string{"woo"}},
				},
			},
		},
		WithAssetService(assetServiceMock{
			assets: map[string]*assetsdomain.Asset{
				"asset-1": {ID: "asset-1", Key: "assets/a-1.jpg"},
				"asset-2": {ID: "asset-2", Metadata: map[string]string{"falabella_url": "https://cdn.example.com/custom.jpg"}},
			},
		}),
		WithAssetBaseURL("https://cdn.example.com"),
	)
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}

	entity, getErr := catalog.GetProduct(context.Background(), "p-1")
	if getErr != nil {
		t.Fatalf("GetProduct() error = %v", getErr)
	}
	if len(entity.Images) != 2 {
		t.Fatalf("len(entity.Images) = %d, want %d", len(entity.Images), 2)
	}
	if entity.Images[0].URL != "https://cdn.example.com/assets/a-1.jpg" {
		t.Fatalf("entity.Images[0].URL = %q, want %q", entity.Images[0].URL, "https://cdn.example.com/assets/a-1.jpg")
	}
	if entity.Images[1].URL != "https://cdn.example.com/custom.jpg" {
		t.Fatalf("entity.Images[1].URL = %q, want %q", entity.Images[1].URL, "https://cdn.example.com/custom.jpg")
	}
	if len(entity.Images[1].VariationIDs) != 1 || entity.Images[1].VariationIDs[0] != "v-color" {
		t.Fatalf("entity.Images[1].VariationIDs = %#v, want [v-color]", entity.Images[1].VariationIDs)
	}
}

// TestCatalogAssetLookupError verifies asset lookup error propagation behavior.
func TestCatalogAssetLookupError(t *testing.T) {
	catalog, err := NewCatalog(
		serviceMock{
			entity: &productdomain.Product{
				ID:      "p-1",
				SKU:     "SKU-1",
				Gallery: []productdomain.GalleryItem{{AssetID: "asset-1"}},
			},
		},
		WithAssetService(assetServiceMock{err: errorspkg.New("boom")}),
	)
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}

	if _, getErr := catalog.GetProduct(context.Background(), "p-1"); getErr == nil {
		t.Fatalf("GetProduct() expected asset lookup error")
	}
}

// TestMapProductNil verifies nil-mapping behavior.
func TestMapProductNil(t *testing.T) {
	entity, err := (&Catalog{}).mapProduct(context.Background(), nil)
	if err != nil {
		t.Fatalf("mapProduct(nil) error = %v", err)
	}
	if entity != nil {
		t.Fatalf("mapProduct(nil) = %#v, want nil", entity)
	}
}
