package domain

import (
	"errors"
	"testing"
)

// TestOrderNormalize validates order normalization behavior.
func TestOrderNormalize(t *testing.T) {
	entity := &Order{
		Identifier: " ORD-1 ",
		Realm:      " woocommerce ",
		ContactID:  " c-1 ",
		Items: []Item{
			{SKU: " SKU-1 ", AlternateName: " Name ", ProductID: " p-1 ", ResolutionSource: " sku "},
		},
		CurrentStatus: " CREATED ",
		StatusHistory: []StatusEntry{
			{Status: " CREATED ", Author: " system ", Description: " created "},
		},
		ShippingAddress: ShippingAddress{
			Address:  " Street 1 ",
			Address2: " Apt 1 ",
			Phone:    " +57 3000000000 ",
			CityCode: " 110111 ",
		},
	}

	entity.Normalize()

	if entity.Identifier != "ORD-1" {
		t.Fatalf("Identifier = %q, want %q", entity.Identifier, "ORD-1")
	}
	if entity.Items[0].SKU != "SKU-1" {
		t.Fatalf("Items[0].SKU = %q, want %q", entity.Items[0].SKU, "SKU-1")
	}
	if entity.CurrentStatus != StatusCreated {
		t.Fatalf("CurrentStatus = %q, want %q", entity.CurrentStatus, StatusCreated)
	}
	if entity.ShippingAddress.CityCode != "110111" {
		t.Fatalf("ShippingAddress.CityCode = %q, want %q", entity.ShippingAddress.CityCode, "110111")
	}
}

// TestOrderValidate validates domain invariant behavior.
func TestOrderValidate(t *testing.T) {
	valid := Order{
		Identifier: "ORD-1",
		Realm:      "woocommerce",
		ContactID:  "c-1",
		Items: []Item{
			{SKU: "SKU-1", Quantity: 1},
		},
		CurrentStatus: StatusCreated,
		StatusHistory: []StatusEntry{
			{Status: StatusCreated, Author: "system"},
		},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	cases := []struct {
		name string
		item Order
		err  error
	}{
		{name: "identifier", item: Order{Realm: "x", ContactID: "c", Items: []Item{{SKU: "sku", Quantity: 1}}, CurrentStatus: StatusCreated}, err: ErrIdentifierRequired},
		{name: "realm", item: Order{Identifier: "i", ContactID: "c", Items: []Item{{SKU: "sku", Quantity: 1}}, CurrentStatus: StatusCreated}, err: ErrRealmRequired},
		{name: "contact", item: Order{Identifier: "i", Realm: "x", Items: []Item{{SKU: "sku", Quantity: 1}}, CurrentStatus: StatusCreated}, err: ErrContactIDRequired},
		{name: "items", item: Order{Identifier: "i", Realm: "x", ContactID: "c", CurrentStatus: StatusCreated}, err: ErrItemsRequired},
		{name: "item sku", item: Order{Identifier: "i", Realm: "x", ContactID: "c", Items: []Item{{Quantity: 1}}, CurrentStatus: StatusCreated}, err: ErrItemSKURequired},
		{name: "item qty", item: Order{Identifier: "i", Realm: "x", ContactID: "c", Items: []Item{{SKU: "s", Quantity: 0}}, CurrentStatus: StatusCreated}, err: ErrItemQuantityInvalid},
		{name: "status", item: Order{Identifier: "i", Realm: "x", ContactID: "c", Items: []Item{{SKU: "s", Quantity: 1}}, CurrentStatus: "bad"}, err: ErrStatusInvalid},
		{name: "status author", item: Order{Identifier: "i", Realm: "x", ContactID: "c", Items: []Item{{SKU: "s", Quantity: 1}}, CurrentStatus: StatusCreated, StatusHistory: []StatusEntry{{Status: StatusCreated}}}, err: ErrStatusAuthorRequired},
	}

	for _, testCase := range cases {
		err := testCase.item.Validate()
		if !errors.Is(err, testCase.err) {
			t.Fatalf("%s: Validate() error = %v, want %v", testCase.name, err, testCase.err)
		}
	}
}
