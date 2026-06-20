package main

import (
	"context"
	"testing"

	ordersdomain "mannaiah/module/orders/domain"
	productdomain "mannaiah/module/products/domain/product"
	variationdomain "mannaiah/module/products/domain/variation"
)

// TestShippingBatchManifestItemLabelsIncludesQuantity verifies PDF summary item labels include quantities.
func TestShippingBatchManifestItemLabelsIncludesQuantity(t *testing.T) {
	labels := shippingBatchManifestItemLabels([]ordersdomain.Item{
		{AlternateName: "Totepack Kairos Classic NEGRO", Quantity: 2},
		{SKU: "SKU-1", Quantity: 0},
	})

	if len(labels) != 2 {
		t.Fatalf("labels len = %d, want 2", len(labels))
	}
	if labels[0] != "X2 Totepack Kairos Classic NEGRO" {
		t.Fatalf("labels[0] = %q", labels[0])
	}
	if labels[1] != "X1 SKU-1" {
		t.Fatalf("labels[1] = %q", labels[1])
	}
}

// TestShippingBatchManifestItemLabelsAppendColorVariationFromSKU verifies SKU-linked product variations enrich PDF labels.
func TestShippingBatchManifestItemLabelsAppendColorVariationFromSKU(t *testing.T) {
	adapter := shippingBatchManifestOrderSummaryAdapter{
		products: shippingManifestProductLookupStub{products: map[string]*productdomain.Product{
			"7709334399073": {
				SKU: "base-sku",
				Variants: []productdomain.Variant{
					{SKU: "7709334399073", VariationIDs: []string{"variation-negro"}},
				},
			},
		}},
		variations: shippingManifestVariationLookupStub{variations: map[string]*variationdomain.Variation{
			"variation-negro": {ID: "variation-negro", Name: "Negro", Definition: variationdomain.DefinitionColor, Value: "#000000"},
		}},
	}

	labels := adapter.itemLabels(context.Background(), []ordersdomain.Item{
		{SKU: "7709334399073", AlternateName: "Totepack Kairos Classic", Quantity: 1},
	})

	if len(labels) != 1 {
		t.Fatalf("labels len = %d, want 1", len(labels))
	}
	if labels[0] != "X1 Totepack Kairos Classic NEGRO" {
		t.Fatalf("labels[0] = %q, want color-enriched label", labels[0])
	}
}

// TestShippingBatchManifestItemLabelsDoNotDuplicateExistingColor verifies labels already containing the color are unchanged.
func TestShippingBatchManifestItemLabelsDoNotDuplicateExistingColor(t *testing.T) {
	adapter := shippingBatchManifestOrderSummaryAdapter{
		products: shippingManifestProductLookupStub{products: map[string]*productdomain.Product{
			"7709334399073": {
				SKU:      "base-sku",
				Variants: []productdomain.Variant{{SKU: "7709334399073", VariationIDs: []string{"variation-negro"}}},
			},
		}},
		variations: shippingManifestVariationLookupStub{variations: map[string]*variationdomain.Variation{
			"variation-negro": {ID: "variation-negro", Name: "Negro", Definition: variationdomain.DefinitionColor, Value: "#000000"},
		}},
	}

	labels := adapter.itemLabels(context.Background(), []ordersdomain.Item{
		{SKU: "7709334399073", AlternateName: "Totepack Kairos Classic Negro", Quantity: 1},
	})

	if labels[0] != "X1 Totepack Kairos Classic Negro" {
		t.Fatalf("labels[0] = %q, want unchanged existing color label", labels[0])
	}
}

type shippingManifestProductLookupStub struct {
	products map[string]*productdomain.Product
}

// GetBySKU resolves product fixtures by SKU.
func (s shippingManifestProductLookupStub) GetBySKU(ctx context.Context, sku string) (*productdomain.Product, error) {
	_ = ctx
	if s.products == nil {
		return nil, nil
	}
	return s.products[sku], nil
}

type shippingManifestVariationLookupStub struct {
	variations map[string]*variationdomain.Variation
}

// Get resolves variation fixtures by ID.
func (s shippingManifestVariationLookupStub) Get(ctx context.Context, id string) (*variationdomain.Variation, error) {
	_ = ctx
	if s.variations == nil {
		return nil, nil
	}
	return s.variations[id], nil
}
