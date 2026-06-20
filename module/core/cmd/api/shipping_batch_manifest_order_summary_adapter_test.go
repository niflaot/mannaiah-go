package main

import (
	"testing"

	ordersdomain "mannaiah/module/orders/domain"
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
