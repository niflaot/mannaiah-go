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
			{Author: "", Description: "note", Internal: true, OccurredAt: time.Date(2026, time.February, 14, 14, 0, 0, 0, time.UTC)},
			{Author: "user", Description: "", OccurredAt: time.Now().UTC()},
		},
	})

	if len(comments) != 1 {
		t.Fatalf("len(comments) = %d, want 1", len(comments))
	}
	if comments[0].Author != syncCommentAuthor {
		t.Fatalf("comments[0].Author = %q, want %q", comments[0].Author, syncCommentAuthor)
	}
	if comments[0].Comment != "note" {
		t.Fatalf("comments[0].Comment = %q, want %q", comments[0].Comment, "note")
	}
	if !comments[0].Internal {
		t.Fatalf("comments[0].Internal = false, want true")
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

// TestMapOrderToContactSyncCommandMapsCheckerMetadata verifies checker metadata propagation behavior.
func TestMapOrderToContactSyncCommandMapsCheckerMetadata(t *testing.T) {
	command, ok, reason := mapOrderToContactSyncCommand(port.WooOrder{
		ID:               1001,
		CreatedAt:        time.Date(2026, time.March, 13, 18, 5, 22, 0, time.UTC),
		BillingEmail:     "consent@example.com",
		BillingFirstName: "Consent",
		BillingLastName:  "User",
		Metadata: map[string]string{
			"flock_checker_privacy_accept":                 "yes",
			"flock_checker_privacy_accept_accepted_at":     "2026-03-13 13:05:22",
			"flock_checker_privacy_accept_accepted_at_utc": "2026-03-13T18:05:22Z",
			"flock_checker_circle_optin":                   "yes",
			"flock_checker_circle_optin_accepted_at":       "2026-03-13 13:05:22",
			"flock_checker_circle_optin_accepted_at_utc":   "2026-03-13T18:05:22Z",
			"flock_checker_terminos_extra":                 "no",
			"flock_checker_terminos_extra_accepted_at":     "2026-03-13 13:05:22",
			"flock_checker_terminos_extra_accepted_at_utc": "2026-03-13T18:05:22Z",
		},
	})
	if !ok {
		t.Fatalf("expected mapped command")
	}
	if reason != "" {
		t.Fatalf("reason = %q, want empty", reason)
	}
	if command.Metadata["flock_checker_privacy_accept"] != "yes" {
		t.Fatalf("privacy checker metadata = %q, want %q", command.Metadata["flock_checker_privacy_accept"], "yes")
	}
	if command.Metadata["flock_checker_privacy_accept_accepted_at"] != "2026-03-13 13:05:22" {
		t.Fatalf("privacy accepted_at metadata = %q, want %q", command.Metadata["flock_checker_privacy_accept_accepted_at"], "2026-03-13 13:05:22")
	}
	if command.Metadata["flock_checker_privacy_accept_accepted_at_utc"] != "2026-03-13T18:05:22Z" {
		t.Fatalf("privacy accepted_at_utc metadata = %q, want %q", command.Metadata["flock_checker_privacy_accept_accepted_at_utc"], "2026-03-13T18:05:22Z")
	}
	if command.Metadata["flock_checker_circle_optin"] != "yes" {
		t.Fatalf("circle optin metadata = %q, want %q", command.Metadata["flock_checker_circle_optin"], "yes")
	}
	if command.Metadata["flock_checker_circle_optin_accepted_at"] != "2026-03-13 13:05:22" {
		t.Fatalf("circle accepted_at metadata = %q, want %q", command.Metadata["flock_checker_circle_optin_accepted_at"], "2026-03-13 13:05:22")
	}
	if command.Metadata["flock_checker_circle_optin_accepted_at_utc"] != "2026-03-13T18:05:22Z" {
		t.Fatalf("circle accepted_at_utc metadata = %q, want %q", command.Metadata["flock_checker_circle_optin_accepted_at_utc"], "2026-03-13T18:05:22Z")
	}
	if command.Metadata["flock_checker_terminos_extra"] != "no" {
		t.Fatalf("terminos_extra metadata = %q, want %q", command.Metadata["flock_checker_terminos_extra"], "no")
	}
	if command.Metadata["flock_checker_terminos_extra_accepted_at"] != "2026-03-13 13:05:22" {
		t.Fatalf("terminos_extra accepted_at metadata = %q, want %q", command.Metadata["flock_checker_terminos_extra_accepted_at"], "2026-03-13 13:05:22")
	}
	if command.Metadata["flock_checker_terminos_extra_accepted_at_utc"] != "2026-03-13T18:05:22Z" {
		t.Fatalf("terminos_extra accepted_at_utc metadata = %q, want %q", command.Metadata["flock_checker_terminos_extra_accepted_at_utc"], "2026-03-13T18:05:22Z")
	}
}

// TestBuildContactMetadataBackfillsCircleOptInAcceptedAt verifies circle opt-in accepted-at fallback behavior.
func TestBuildContactMetadataBackfillsCircleOptInAcceptedAt(t *testing.T) {
	createdAt := time.Date(2026, time.March, 13, 18, 5, 22, 0, time.UTC)
	metadata := buildContactMetadata(port.WooOrder{
		Metadata: map[string]string{
			"flock_checker_circle_optin": "yes",
		},
	}, &createdAt)

	if metadata["flock_checker_circle_optin"] != "yes" {
		t.Fatalf("circle optin metadata = %q, want %q", metadata["flock_checker_circle_optin"], "yes")
	}
	if metadata["flock_checker_circle_optin_accepted_at"] != "2026-03-13 13:05:22" {
		t.Fatalf("circle accepted_at metadata = %q, want %q", metadata["flock_checker_circle_optin_accepted_at"], "2026-03-13 13:05:22")
	}
	if metadata["flock_checker_circle_optin_accepted_at_utc"] != "2026-03-13T18:05:22Z" {
		t.Fatalf("circle accepted_at_utc metadata = %q, want %q", metadata["flock_checker_circle_optin_accepted_at_utc"], "2026-03-13T18:05:22Z")
	}
}

// TestBuildContactMetadataMapsCircleOptOutToRejectedAt verifies circle opt-out metadata mapping behavior.
func TestBuildContactMetadataMapsCircleOptOutToRejectedAt(t *testing.T) {
	createdAt := time.Date(2026, time.March, 13, 18, 5, 22, 0, time.UTC)
	metadata := buildContactMetadata(port.WooOrder{
		Metadata: map[string]string{
			"flock_checker_circle_optin":             "no",
			"flock_checker_circle_optin_accepted_at": "2026-03-13 13:05:22",
		},
	}, &createdAt)

	if metadata["flock_checker_circle_optin"] != "no" {
		t.Fatalf("circle optin metadata = %q, want %q", metadata["flock_checker_circle_optin"], "no")
	}
	if metadata["flock_checker_circle_optin_rejected_at"] != "2026-03-13 13:05:22" {
		t.Fatalf("circle rejected_at metadata = %q, want %q", metadata["flock_checker_circle_optin_rejected_at"], "2026-03-13 13:05:22")
	}
	if metadata["flock_checker_circle_optin_rejected_at_utc"] != "2026-03-13T18:05:22Z" {
		t.Fatalf("circle rejected_at_utc metadata = %q, want %q", metadata["flock_checker_circle_optin_rejected_at_utc"], "2026-03-13T18:05:22Z")
	}
	if _, exists := metadata["flock_checker_circle_optin_accepted_at"]; exists {
		t.Fatalf("expected circle accepted_at metadata to be cleared for no decision")
	}
}
