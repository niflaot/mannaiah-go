package woocommerce

import (
	"testing"

	wcentity "github.com/jmolboy/woocommerce-go/entity"
)

// TestMapRawOrderMapsPaymentMethod verifies payment_method field mapping in raw order decode.
func TestMapRawOrderMapsPaymentMethod(t *testing.T) {
	order := mapRawOrder(rawOrderPayload{
		ID:            1001,
		Status:        "processing",
		PaymentMethod: "  cod  ",
		Billing: struct {
			Email     string `json:"email"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Company   string `json:"company"`
			Phone     string `json:"phone"`
			Address1  string `json:"address_1"`
			Address2  string `json:"address_2"`
			City      string `json:"city"`
		}{
			Email:     "raw@example.com",
			FirstName: "Raw",
			LastName:  "Test",
		},
	})
	if order.PaymentMethod != "cod" {
		t.Fatalf("PaymentMethod = %q, want %q", order.PaymentMethod, "cod")
	}
}

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

// TestMapSDKFeeItemsIncludesQuotaRows verifies SDK fee-line mapping behavior for non-product order lines.
func TestMapSDKFeeItemsIncludesQuotaRows(t *testing.T) {
	items := mapSDKFeeItems([]wcentity.FeeLine{
		{Name: "Cuotas", Total: 137000},
		{Name: " ", Total: 1000},
	})

	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Name != "Cuotas" || items[0].Quantity != 1 || items[0].Value != 137000 {
		t.Fatalf("items[0] = %+v, want mapped fee line", items[0])
	}
}

// TestMapRawFeeItemsIncludesQuotaRows verifies raw fee-line mapping behavior for non-product order lines.
func TestMapRawFeeItemsIncludesQuotaRows(t *testing.T) {
	items := mapRawFeeItems([]rawFeeLine{
		{Name: "Cuotas", Total: 137000},
		{Name: " ", Total: 1000},
	})

	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Name != "Cuotas" || items[0].Quantity != 1 || items[0].Value != 137000 {
		t.Fatalf("items[0] = %+v, want mapped fee line", items[0])
	}
}
