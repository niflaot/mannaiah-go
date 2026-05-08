package service

import (
	"testing"
	"time"

	contactsdomain "mannaiah/module/contacts/domain"
	shopifyport "mannaiah/module/shopify/port"
)

func TestBuildContactSyncCommand_CleanNumericCompanyIsDocumentNumber(t *testing.T) {
	customer := shopifyport.ShopifyCustomer{
		ID:         "111",
		ShopDomain: "shop.myshopify.com",
		Email:      "test@example.com",
		FirstName:  "Ana",
		LastName:   "García",
		DefaultAddress: &shopifyport.ShopifyAddress{
			Company: "12345678",
		},
	}

	cmd := BuildContactSyncCommand(customer)

	if cmd.DocumentNumber != "12345678" {
		t.Errorf("DocumentNumber = %q, want %q", cmd.DocumentNumber, "12345678")
	}
	if cmd.DocumentType != contactsdomain.DocumentTypeCC {
		t.Errorf("DocumentType = %q, want CC", cmd.DocumentType)
	}
}

func TestBuildContactSyncCommand_FormattedCompanyStripsNonDigits(t *testing.T) {
	cases := []struct {
		company string
		want    string
	}{
		{"1.234.567", "1234567"},
		{"1,234,567", "1234567"},
		{"12 345 678", "12345678"},
		{"12-345-678", "12345678"},
		{"  1.234.1521  ", "12341521"},
		{"10.203.040-5", "102030405"},
	}

	for _, tc := range cases {
		customer := shopifyport.ShopifyCustomer{
			ID:    "999",
			Email: "x@example.com",
			DefaultAddress: &shopifyport.ShopifyAddress{
				Company: tc.company,
			},
		}
		cmd := BuildContactSyncCommand(customer)
		if cmd.DocumentNumber != tc.want {
			t.Errorf("company %q → DocumentNumber = %q, want %q", tc.company, cmd.DocumentNumber, tc.want)
		}
		if cmd.DocumentType != contactsdomain.DocumentTypeCC {
			t.Errorf("company %q → DocumentType = %q, want CC", tc.company, cmd.DocumentType)
		}
	}
}

func TestBuildContactSyncCommand_BlankCompanyProducesNoDocument(t *testing.T) {
	cases := []string{"", "   ", "-", ".", "N/A"}

	for _, company := range cases {
		customer := shopifyport.ShopifyCustomer{
			ID:    "222",
			Email: "nodoc@example.com",
			DefaultAddress: &shopifyport.ShopifyAddress{
				Company: company,
			},
		}
		cmd := BuildContactSyncCommand(customer)
		if cmd.DocumentNumber != "" {
			t.Errorf("company %q → DocumentNumber = %q, want empty", company, cmd.DocumentNumber)
		}
		if cmd.DocumentType != "" {
			t.Errorf("company %q → DocumentType = %q, want empty", company, cmd.DocumentType)
		}
	}
}

func TestBuildContactSyncCommand_NoDefaultAddressProducesNoDocument(t *testing.T) {
	customer := shopifyport.ShopifyCustomer{
		ID:    "333",
		Email: "noaddr@example.com",
	}

	cmd := BuildContactSyncCommand(customer)

	if cmd.DocumentNumber != "" {
		t.Errorf("DocumentNumber = %q, want empty when no default address", cmd.DocumentNumber)
	}
	if cmd.DocumentType != "" {
		t.Errorf("DocumentType = %q, want empty when no default address", cmd.DocumentType)
	}
}

func TestBuildContactSyncCommand_NoteAttributesAreIgnored(t *testing.T) {
	// NoteAttributes must NOT influence document extraction — Company is authoritative.
	customer := shopifyport.ShopifyCustomer{
		ID:    "444",
		Email: "notes@example.com",
		NoteAttributes: []shopifyport.ShopifyNoteAttribute{
			{Name: "document_type", Value: "CE"},
			{Name: "document_number", Value: "99887766"},
		},
		DefaultAddress: &shopifyport.ShopifyAddress{
			Company: "11223344",
		},
	}

	cmd := BuildContactSyncCommand(customer)

	if cmd.DocumentNumber != "11223344" {
		t.Errorf("DocumentNumber = %q, want %q (company wins, not note_attributes)", cmd.DocumentNumber, "11223344")
	}
	if cmd.DocumentType != contactsdomain.DocumentTypeCC {
		t.Errorf("DocumentType = %q, want CC (always for Shopify e-commerce)", cmd.DocumentType)
	}
}

func TestBuildContactSyncCommand_NameFallsBackToDefaultAddress(t *testing.T) {
	customer := shopifyport.ShopifyCustomer{
		ID:    "555",
		Email: "namefb@example.com",
		DefaultAddress: &shopifyport.ShopifyAddress{
			Company:   "87654321",
			FirstName: "Carlos",
			LastName:  "Mejía",
		},
	}

	cmd := BuildContactSyncCommand(customer)

	if cmd.FirstName != "Carlos" {
		t.Errorf("FirstName = %q, want %q from address", cmd.FirstName, "Carlos")
	}
	if cmd.LastName != "Mejía" {
		t.Errorf("LastName = %q, want %q from address", cmd.LastName, "Mejía")
	}
}

func TestBuildContactSyncCommand_PhoneAndAddressFromDefaultAddress(t *testing.T) {
	createdAt := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	customer := shopifyport.ShopifyCustomer{
		ID:        "666",
		Email:     "addr@example.com",
		FirstName: "Pedro",
		LastName:  "Rodríguez",
		CreatedAt: createdAt,
		DefaultAddress: &shopifyport.ShopifyAddress{
			Company:  "55556666",
			Address1: "Calle 80 # 23-45",
			Address2: "Apto 101",
			City:     "Bogotá",
			Phone:    "3001112233",
		},
	}

	cmd := BuildContactSyncCommand(customer)

	if cmd.Address != "Calle 80 # 23-45" {
		t.Errorf("Address = %q, want %q", cmd.Address, "Calle 80 # 23-45")
	}
	if cmd.AddressExtra != "Apto 101" {
		t.Errorf("AddressExtra = %q, want %q", cmd.AddressExtra, "Apto 101")
	}
	if cmd.CityCode != "11001" {
		t.Errorf("CityCode = %q, want %q", cmd.CityCode, "11001")
	}
	if cmd.Phone != "3001112233" {
		t.Errorf("Phone = %q, want %q", cmd.Phone, "3001112233")
	}
	if cmd.CreatedAt == nil || !cmd.CreatedAt.Equal(createdAt) {
		t.Errorf("CreatedAt = %v, want %v", cmd.CreatedAt, createdAt)
	}
}

// TestBuildContactSyncCommandMapsMarketingConsent verifies Shopify checkout opt-in metadata is preserved.
func TestBuildContactSyncCommandMapsMarketingConsent(t *testing.T) {
	consentedAt := time.Date(2026, time.May, 6, 22, 0, 0, 0, time.UTC)
	customer := shopifyport.ShopifyCustomer{
		ID:                             "777",
		Email:                          "optin@example.com",
		FirstName:                      "Opt",
		LastName:                       "In",
		EmailMarketingState:            "subscribed",
		EmailMarketingConsentUpdatedAt: &consentedAt,
		SMSMarketingState:              "not_subscribed",
		SMSMarketingConsentUpdatedAt:   nil,
	}

	cmd := BuildContactSyncCommand(customer)

	if cmd.Metadata["membership.opt_in"] != "true" {
		t.Fatalf("membership.opt_in = %q, want true", cmd.Metadata["membership.opt_in"])
	}
	if cmd.Metadata["membership.opt_in_date"] != "2026-05-06T22:00:00Z" {
		t.Fatalf("membership.opt_in_date = %q, want consent timestamp", cmd.Metadata["membership.opt_in_date"])
	}
	if cmd.Metadata["shopify_email_marketing_state"] != "subscribed" {
		t.Fatalf("shopify_email_marketing_state = %q, want subscribed", cmd.Metadata["shopify_email_marketing_state"])
	}
	if cmd.Metadata["shopify_sms_marketing_state"] != "not_subscribed" {
		t.Fatalf("shopify_sms_marketing_state = %q, want not_subscribed", cmd.Metadata["shopify_sms_marketing_state"])
	}
}
