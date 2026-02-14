package service

import (
	"testing"
	"time"

	"mannaiah/module/woocommerce/port"
)

// TestMapOrderToCommandFallbackIdentifier verifies metadata fallback identifier behavior.
func TestMapOrderToCommandFallbackIdentifier(t *testing.T) {
	command, ok, reason := mapOrderToCommand(port.WooOrder{
		ID:               0,
		Status:           "processing",
		BillingEmail:     "fallback@example.com",
		BillingFirstName: "Fall",
		BillingLastName:  "Back",
		BillingAddress1:  "A",
		BillingCity:      "11001",
		Items:            []port.WooOrderItem{{SKU: "SKU-1", Quantity: 1}},
		Metadata: map[string]string{
			"integration.woocommerce.order_id": "1002",
		},
	})
	if !ok {
		t.Fatalf("expected mapped command")
	}
	if reason != "" {
		t.Fatalf("reason = %q, want empty", reason)
	}
	if command.Identifier != "1002" {
		t.Fatalf("command.Identifier = %q, want %q", command.Identifier, "1002")
	}
	if command.Metadata != nil {
		t.Fatalf("command.Metadata = %+v, want nil", command.Metadata)
	}
}

// TestMapOrderToCommandRejectsInvalid verifies invalid order mapping rejection behavior.
func TestMapOrderToCommandRejectsInvalid(t *testing.T) {
	if _, ok, reason := mapOrderToCommand(port.WooOrder{
		ID:    10,
		Items: []port.WooOrderItem{{SKU: "SKU-1", Quantity: 1}},
	}); ok {
		t.Fatalf("expected rejection for empty billing email")
	} else if reason != skipReasonMissingContactEmail {
		t.Fatalf("reason = %q, want %q", reason, skipReasonMissingContactEmail)
	}
	if _, ok, reason := mapOrderToCommand(port.WooOrder{
		ID:               10,
		BillingEmail:     "x@example.com",
		BillingFirstName: "No",
		BillingLastName:  "Items",
	}); ok {
		t.Fatalf("expected rejection for empty order items")
	} else if reason != skipReasonMissingSupportedItems {
		t.Fatalf("reason = %q, want %q", reason, skipReasonMissingSupportedItems)
	}
}

// TestMapOrderItemsAcceptsQuotaRows verifies non-SKU item mapping behavior.
func TestMapOrderItemsAcceptsQuotaRows(t *testing.T) {
	items := mapOrderItems([]port.WooOrderItem{
		{SKU: "", Name: "Cuota 1/3", Quantity: 1},
		{SKU: "", Name: "", Quantity: 1},
	})

	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].SKU != "" || items[0].Name != "Cuota 1/3" {
		t.Fatalf("items[0] = %+v, want quota row mapped by name", items[0])
	}
	if items[0].Quantity != 1 {
		t.Fatalf("items[0].Quantity = %d, want 1", items[0].Quantity)
	}
}

// TestMapOrderCommentsDefaultAuthor verifies default comment author mapping behavior.
func TestMapOrderCommentsDefaultAuthor(t *testing.T) {
	comments := mapOrderComments(port.WooOrder{
		Comments: []port.WooOrderComment{
			{Author: "", Description: "note", OccurredAt: time.Date(2026, time.February, 14, 14, 0, 0, 0, time.UTC)},
			{Author: "user", Description: "", OccurredAt: time.Now().UTC()},
		},
	})

	if len(comments) != 1 {
		t.Fatalf("len(comments) = %d, want 1", len(comments))
	}
	if comments[0].Owner != syncNoteOwner {
		t.Fatalf("comments[0].Owner = %q, want %q", comments[0].Owner, syncNoteOwner)
	}
	if comments[0].Note != "note" {
		t.Fatalf("comments[0].Note = %q, want %q", comments[0].Note, "note")
	}
}

// TestMapShippingAddressFallsBackToBilling verifies billing snapshot fallback behavior.
func TestMapShippingAddressFallsBackToBilling(t *testing.T) {
	address := mapShippingAddress(port.WooOrder{
		BillingAddress1: "Billing 1",
		BillingAddress2: "Billing 2",
		BillingPhone:    "3000000000",
		BillingCity:     "05001",
	})
	if address == nil {
		t.Fatalf("expected billing fallback address")
	}
	if address.Address != "Billing 1" || address.CityCode != "05001" {
		t.Fatalf("address = %+v, want billing snapshot values", address)
	}
}
