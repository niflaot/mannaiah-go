package service

import (
	"testing"
	"time"

	contactsdomain "mannaiah/module/contacts/domain"
	ordersdomain "mannaiah/module/orders/domain"
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

// TestBuildOrderSyncCommandMapsPaidUnfulfilledToCreated verifies fulfillment state remains operationally authoritative.
func TestBuildOrderSyncCommandMapsPaidUnfulfilledToCreated(t *testing.T) {
	command := BuildOrderSyncCommand(shopifyport.ShopifyOrder{
		ID:                "order-1",
		Name:              "#1001",
		FinancialStatus:   "paid",
		FulfillmentStatus: "",
	}, "contact-1", "shopify", "cron")

	if command.Status != ordersdomain.StatusCreated {
		t.Fatalf("Status = %q, want CREATED for paid + unfulfilled", command.Status)
	}
	if command.Source != syncMutationSource {
		t.Fatalf("Source = %q, want %q", command.Source, syncMutationSource)
	}
}

// TestBuildOrderSyncCommandMapsPaidFulfilledToCompleted verifies completed remains tied to fulfillment.
func TestBuildOrderSyncCommandMapsPaidFulfilledToCompleted(t *testing.T) {
	command := BuildOrderSyncCommand(shopifyport.ShopifyOrder{
		ID:                "order-1",
		Name:              "#1001",
		FinancialStatus:   "paid",
		FulfillmentStatus: "fulfilled",
	}, "contact-1", "shopify", "cron")

	if command.Status != ordersdomain.StatusCompleted {
		t.Fatalf("Status = %q, want COMPLETED for fulfilled order", command.Status)
	}
}

// TestBuildOrderSyncCommandMapsPendingCODToCreated verifies COD imports are operationally processable.
func TestBuildOrderSyncCommandMapsPendingCODToCreated(t *testing.T) {
	command := BuildOrderSyncCommand(shopifyport.ShopifyOrder{
		ID:                  "order-1",
		Name:                "#1001",
		FinancialStatus:     "pending",
		PaymentGatewayNames: []string{"Cash on Delivery (COD)"},
	}, "contact-1", "shopify", "webhook")

	if command.Status != ordersdomain.StatusCreated {
		t.Fatalf("Status = %q, want CREATED for pending COD", command.Status)
	}
}

// TestBuildOrderSyncCommandKeepsPendingNonCODPending verifies non-COD unpaid orders remain blocked.
func TestBuildOrderSyncCommandKeepsPendingNonCODPending(t *testing.T) {
	command := BuildOrderSyncCommand(shopifyport.ShopifyOrder{
		ID:                  "order-1",
		Name:                "#1001",
		FinancialStatus:     "pending",
		PaymentGatewayNames: []string{"Addi Payment"},
	}, "contact-1", "shopify", "webhook")

	if command.Status != ordersdomain.StatusPending {
		t.Fatalf("Status = %q, want PENDING for pending non-COD", command.Status)
	}
}

// TestBuildOrderSyncCommandMapsLineItemProductIdentity verifies Shopify line identity survives sync mapping.
func TestBuildOrderSyncCommandMapsLineItemProductIdentity(t *testing.T) {
	command := BuildOrderSyncCommand(shopifyport.ShopifyOrder{
		ID:   "order-1",
		Name: "#1001",
		LineItems: []shopifyport.ShopifyLineItem{
			{
				SKU:               "7700001",
				Title:             "Morral",
				ProductID:         "gid://shopify/Product/1",
				VariantID:         "gid://shopify/ProductVariant/2",
				MannaiahProductID: "product-1",
				Quantity:          1,
				Price:             "120000",
			},
		},
	}, "contact-1", "shopify", "cron")

	if len(command.Items) != 1 {
		t.Fatalf("Items len = %d, want 1", len(command.Items))
	}
	item := command.Items[0]
	if item.ProductID != "product-1" {
		t.Fatalf("ProductID = %q, want product-1", item.ProductID)
	}
	if item.ShopifyProductID != "gid://shopify/Product/1" {
		t.Fatalf("ShopifyProductID = %q, want Shopify product GID", item.ShopifyProductID)
	}
	if item.ShopifyVariantID != "gid://shopify/ProductVariant/2" {
		t.Fatalf("ShopifyVariantID = %q, want Shopify variant GID", item.ShopifyVariantID)
	}
}

// TestBuildOrderSyncCommandAppendsProductColorLabel verifies Shopify product-color labels are preserved for PDFs.
func TestBuildOrderSyncCommandAppendsProductColorLabel(t *testing.T) {
	command := BuildOrderSyncCommand(shopifyport.ShopifyOrder{
		ID:   "order-1",
		Name: "#1001",
		LineItems: []shopifyport.ShopifyLineItem{
			{
				SKU:        "7700001",
				Title:      "Totepack Kairos Classic",
				ColorLabel: "Negro",
				Quantity:   1,
				Price:      "120000",
			},
		},
	}, "contact-1", "shopify", "cron")

	if command.Items[0].AlternateName != "Totepack Kairos Classic NEGRO" {
		t.Fatalf("AlternateName = %q, want color suffix", command.Items[0].AlternateName)
	}
}

// TestBuildOrderSyncCommandDoesNotDuplicateProductColorLabel verifies existing color names are not repeated.
func TestBuildOrderSyncCommandDoesNotDuplicateProductColorLabel(t *testing.T) {
	command := BuildOrderSyncCommand(shopifyport.ShopifyOrder{
		ID:   "order-1",
		Name: "#1001",
		LineItems: []shopifyport.ShopifyLineItem{
			{SKU: "7700001", Title: "Totepack Kairos Classic Negro", ColorLabel: "Negro", Quantity: 1},
		},
	}, "contact-1", "shopify", "cron")

	if command.Items[0].AlternateName != "Totepack Kairos Classic Negro" {
		t.Fatalf("AlternateName = %q, want unchanged label", command.Items[0].AlternateName)
	}
}

// TestBuildOrderSyncCommandMapsShippingCityNameToCityCode verifies Shopify city names are normalized for Mannaiah.
func TestBuildOrderSyncCommandMapsShippingCityNameToCityCode(t *testing.T) {
	command := BuildOrderSyncCommand(shopifyport.ShopifyOrder{
		ID:   "order-1",
		Name: "#1001",
		ShippingAddress: &shopifyport.ShopifyAddress{
			Address1: "Calle 80 # 23-45",
			City:     "Bogotá",
		},
	}, "contact-1", "shopify", "cron")

	if command.ShippingAddress == nil {
		t.Fatal("ShippingAddress = nil, want mapped address")
	}
	if command.ShippingAddress.CityCode != "11001" {
		t.Fatalf("ShippingAddress.CityCode = %q, want 11001", command.ShippingAddress.CityCode)
	}
}

// TestBuildOrderSyncCommandUsesProvinceToResolveDuplicatedShippingCity verifies department evidence resolves duplicated city names.
func TestBuildOrderSyncCommandUsesProvinceToResolveDuplicatedShippingCity(t *testing.T) {
	command := BuildOrderSyncCommand(shopifyport.ShopifyOrder{
		ID:   "order-1",
		Name: "#1001",
		ShippingAddress: &shopifyport.ShopifyAddress{
			Address1: "Barrio Santa Rita Mz 5 Casa 5",
			City:     "Armenia",
			Province: "Quindio",
		},
	}, "contact-1", "shopify", "cron")

	if command.ShippingAddress == nil {
		t.Fatal("ShippingAddress = nil, want mapped address")
	}
	if command.ShippingAddress.CityCode != "63001" {
		t.Fatalf("ShippingAddress.CityCode = %q, want 63001", command.ShippingAddress.CityCode)
	}
	if command.Metadata["shipping_city_resolution_status"] != "resolved" {
		t.Fatalf("resolution status = %q, want resolved", command.Metadata["shipping_city_resolution_status"])
	}
}

// TestBuildOrderSyncCommandStoresUnresolvedCityMetadata verifies ambiguous cities leave operator-facing metadata.
func TestBuildOrderSyncCommandStoresUnresolvedCityMetadata(t *testing.T) {
	command := BuildOrderSyncCommand(shopifyport.ShopifyOrder{
		ID:   "order-1",
		Name: "#1001",
		ShippingAddress: &shopifyport.ShopifyAddress{
			Address1: "Barrio Santa Rita Mz 5 Casa 5",
			City:     "Armenia",
		},
	}, "contact-1", "shopify", "cron")

	if command.ShippingAddress == nil {
		t.Fatal("ShippingAddress = nil, want mapped address")
	}
	if command.ShippingAddress.CityCode != "-1" {
		t.Fatalf("ShippingAddress.CityCode = %q, want -1", command.ShippingAddress.CityCode)
	}
	if command.Metadata["shipping_city_resolution_status"] != "unresolved" {
		t.Fatalf("resolution status = %q, want unresolved", command.Metadata["shipping_city_resolution_status"])
	}
	if command.Metadata["shipping_city_resolution_reason"] == "" {
		t.Fatal("resolution reason is empty, want failure reason")
	}
	if command.Metadata["shipping_city_resolution_suggestions"] == "" {
		t.Fatal("resolution suggestions are empty, want operator hints")
	}
}

// TestBuildOrderContactSyncCommandMarksPrivacyFromOrderDate verifies Shopify order creation stamps privacy acceptance.
func TestBuildOrderContactSyncCommandMarksPrivacyFromOrderDate(t *testing.T) {
	createdAt := time.Date(2026, time.May, 6, 21, 22, 51, 0, time.UTC)
	command, err := BuildOrderContactSyncCommand(shopifyport.ShopifyOrder{
		ContactEmail: "buyer@example.com",
		CreatedAt:    createdAt,
		Customer: &shopifyport.ShopifyCustomer{
			ID:        "customer-1",
			FirstName: "Ada",
			LastName:  "Lovelace",
		},
	})
	if err != nil {
		t.Fatalf("BuildOrderContactSyncCommand() error = %v", err)
	}
	if command.Metadata["privacy.accepted"] != "true" {
		t.Fatalf("privacy.accepted = %q, want true", command.Metadata["privacy.accepted"])
	}
	if command.Metadata["privacy.acceptedDate"] != "2026-05-06T21:22:51Z" {
		t.Fatalf("privacy.acceptedDate = %q, want order date", command.Metadata["privacy.acceptedDate"])
	}
}
