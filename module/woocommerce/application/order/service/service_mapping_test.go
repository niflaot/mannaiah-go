package service

import (
	"testing"
	"time"

	"mannaiah/module/woocommerce/port"
)

// TestMapOrderToCommandFallbackIdentifier verifies metadata fallback identifier behavior.
func TestMapOrderToCommandFallbackIdentifier(t *testing.T) {
	command, ok := mapOrderToCommand(port.WooOrder{
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
	if command.Identifier != "1002" {
		t.Fatalf("command.Identifier = %q, want %q", command.Identifier, "1002")
	}
}

// TestMapOrderToCommandRejectsInvalid verifies invalid order mapping rejection behavior.
func TestMapOrderToCommandRejectsInvalid(t *testing.T) {
	if _, ok := mapOrderToCommand(port.WooOrder{
		ID:    10,
		Items: []port.WooOrderItem{{SKU: "SKU-1", Quantity: 1}},
	}); ok {
		t.Fatalf("expected rejection for empty billing email")
	}
	if _, ok := mapOrderToCommand(port.WooOrder{
		ID:               10,
		BillingEmail:     "x@example.com",
		BillingFirstName: "No",
		BillingLastName:  "Items",
	}); ok {
		t.Fatalf("expected rejection for empty order items")
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
	if comments[0].Author != "system" {
		t.Fatalf("comments[0].Author = %q, want %q", comments[0].Author, "system")
	}
}

// TestMergeMetadata verifies metadata merge behavior.
func TestMergeMetadata(t *testing.T) {
	merged := mergeMetadata(
		map[string]string{" left ": " one ", "": "skip"},
		map[string]string{"right": "two"},
	)
	if merged["left"] != "one" || merged["right"] != "two" {
		t.Fatalf("merged metadata = %+v, want normalized values", merged)
	}
}
