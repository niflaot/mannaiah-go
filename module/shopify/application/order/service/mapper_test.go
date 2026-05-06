package service

import (
	"testing"

	contactsdomain "mannaiah/module/contacts/domain"
	shopifyport "mannaiah/module/shopify/port"
)

// TestBuildOrderContactSyncCommandUsesCustomerDefaultAddressCompanyAsDocument verifies document mapping parity with contact sync.
func TestBuildOrderContactSyncCommandUsesCustomerDefaultAddressCompanyAsDocument(t *testing.T) {
	command, err := BuildOrderContactSyncCommand(shopifyport.ShopifyOrder{
		ContactEmail: "buyer@example.com",
		Customer: &shopifyport.ShopifyCustomer{
			ID:        "customer-1",
			FirstName: "Ada",
			LastName:  "Lovelace",
			DefaultAddress: &shopifyport.ShopifyAddress{
				Company: "1.234.567",
			},
		},
		NoteAttributes: []shopifyport.ShopifyNoteAttribute{
			{Name: "document_type", Value: "NIT"},
			{Name: "document_number", Value: "900123456"},
		},
	})
	if err != nil {
		t.Fatalf("BuildOrderContactSyncCommand() error = %v", err)
	}
	if command.DocumentType != contactsdomain.DocumentTypeCC {
		t.Fatalf("DocumentType = %q, want CC", command.DocumentType)
	}
	if command.DocumentNumber != "1234567" {
		t.Fatalf("DocumentNumber = %q, want digits from default address company", command.DocumentNumber)
	}
}

// TestBuildOrderContactSyncCommandIgnoresNoteAttributesWithoutDefaultAddressCompany verifies notes no longer provide documents.
func TestBuildOrderContactSyncCommandIgnoresNoteAttributesWithoutDefaultAddressCompany(t *testing.T) {
	command, err := BuildOrderContactSyncCommand(shopifyport.ShopifyOrder{
		ContactEmail: "buyer@example.com",
		Customer: &shopifyport.ShopifyCustomer{
			ID:        "customer-1",
			FirstName: "Ada",
			LastName:  "Lovelace",
			NoteAttributes: []shopifyport.ShopifyNoteAttribute{
				{Name: "document_type", Value: "CC"},
				{Name: "document_number", Value: "123456789"},
			},
		},
		NoteAttributes: []shopifyport.ShopifyNoteAttribute{
			{Name: "document_type", Value: "CC"},
			{Name: "document_number", Value: "987654321"},
		},
	})
	if err != nil {
		t.Fatalf("BuildOrderContactSyncCommand() error = %v", err)
	}
	if command.DocumentType != "" {
		t.Fatalf("DocumentType = %q, want blank", command.DocumentType)
	}
	if command.DocumentNumber != "" {
		t.Fatalf("DocumentNumber = %q, want blank", command.DocumentNumber)
	}
}
