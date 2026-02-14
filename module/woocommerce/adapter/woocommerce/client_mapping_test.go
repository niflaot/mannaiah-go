package woocommerce

import (
	"testing"

	wcentity "github.com/jmolboy/woocommerce-go/entity"
)

// TestMapSDKOrderItemsIncludesQuotaRows verifies SDK item mapping behavior for non-SKU quota rows.
func TestMapSDKOrderItemsIncludesQuotaRows(t *testing.T) {
	items := mapSDKOrderItems([]wcentity.LineItem{
		{SKU: "SKU-1", Name: "Product", Quantity: 1},
		{SKU: "", Name: "Cuota 1/3", Quantity: 1},
		{SKU: "", Name: "", Quantity: 1},
	})

	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].SKU != "SKU-1" || items[1].Name != "Cuota 1/3" {
		t.Fatalf("items = %+v, want sku and quota rows", items)
	}
}

// TestMapRawOrderItemsIncludesQuotaRows verifies raw item mapping behavior for non-SKU quota rows.
func TestMapRawOrderItemsIncludesQuotaRows(t *testing.T) {
	items := mapRawOrderItems([]rawLineItem{
		{SKU: "SKU-1", Name: "Product", Quantity: 1},
		{SKU: "", Name: "Cuota 2/3", Quantity: 1},
		{SKU: "", Name: "", Quantity: 1},
	})

	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].SKU != "SKU-1" || items[1].Name != "Cuota 2/3" {
		t.Fatalf("items = %+v, want sku and quota rows", items)
	}
}

