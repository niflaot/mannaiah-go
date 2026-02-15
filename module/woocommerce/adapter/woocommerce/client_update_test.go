package woocommerce

import (
	"context"
	"errors"
	"testing"

	"mannaiah/module/woocommerce/port"
)

// TestParseWooOrderID verifies WooCommerce order-id parsing behavior.
func TestParseWooOrderID(t *testing.T) {
	id, err := parseWooOrderID("1001")
	if err != nil {
		t.Fatalf("parseWooOrderID() error = %v", err)
	}
	if id != 1001 {
		t.Fatalf("parseWooOrderID() = %d, want %d", id, 1001)
	}

	if _, err := parseWooOrderID(""); err == nil {
		t.Fatalf("expected error for empty identifier")
	}
	if _, err := parseWooOrderID("abc"); err == nil {
		t.Fatalf("expected error for non-numeric identifier")
	}
}

// TestMapShippingLinesForUpdate verifies shipping-line mapping behavior.
func TestMapShippingLinesForUpdate(t *testing.T) {
	lines := mapShippingLinesForUpdate([]port.OrderSyncShippingCharge{
		{MethodID: "flat_rate", MethodTitle: "Flat", Price: 10},
		{MethodID: "", MethodTitle: "", Price: 0},
	})
	if len(lines) != 1 {
		t.Fatalf("len(lines) = %d, want %d", len(lines), 1)
	}
	if lines[0].MethodId != "flat_rate" {
		t.Fatalf("lines[0].MethodId = %q, want %q", lines[0].MethodId, "flat_rate")
	}
}

// TestMapAddressForUpdate verifies address mapping behavior.
func TestMapAddressForUpdate(t *testing.T) {
	value := port.OrderSyncShippingAddress{
		Address:  "Street 1",
		Address2: "Apt 2",
		Phone:    "300",
		CityCode: "11001",
	}
	shipping := mapShippingAddressForUpdate(value)
	billing := mapBillingAddressForUpdate(value)
	if shipping.City != "11001" {
		t.Fatalf("shipping.City = %q, want %q", shipping.City, "11001")
	}
	if billing.Phone != "300" {
		t.Fatalf("billing.Phone = %q, want %q", billing.Phone, "300")
	}
}

// TestResolveOrderItemsForUpdateFallbackToFeeLines verifies fee-line fallback behavior when SKUs are not resolvable.
func TestResolveOrderItemsForUpdateFallbackToFeeLines(t *testing.T) {
	client := &Client{}
	lineItems, feeLines, err := client.resolveOrderItemsForUpdate(context.Background(), []port.OrderSyncItem{
		{SKU: "", Name: "Quota", Quantity: 1, Value: 20},
	})
	if err != nil {
		t.Fatalf("resolveOrderItemsForUpdate() error = %v", err)
	}
	if len(lineItems) != 0 {
		t.Fatalf("len(lineItems) = %d, want %d", len(lineItems), 0)
	}
	if len(feeLines) != 1 || feeLines[0].Name != "Quota" {
		t.Fatalf("feeLines = %+v, want one Quota line", feeLines)
	}
}

// TestResolveOrderItemsForUpdateCanceledContext verifies context cancellation behavior.
func TestResolveOrderItemsForUpdateCanceledContext(t *testing.T) {
	client := &Client{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := client.resolveOrderItemsForUpdate(ctx, []port.OrderSyncItem{{SKU: "SKU-1", Quantity: 1}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("resolveOrderItemsForUpdate() error = %v, want context.Canceled", err)
	}
}

