package falabella

import (
	"encoding/json"
	"strings"
	"testing"

	"mannaiah/module/falabella/port"
)

// syncProductRequestFixture builds sync-product test fixtures.
func syncProductRequestFixture() port.SyncProductRequest {
	return port.SyncProductRequest{
		SKU:             "SKU-1",
		ParentSKU:       "SKU-PARENT",
		Variation:       "parent",
		Name:            "Backpack Name",
		Brand:           "GENERIC",
		Model:           "M-1",
		Description:     "Backpack description",
		PrimaryCategory: "1638",
		TaxClass:        "19%",
		Price:           "100000",
		SalePrice:       "90000",
		SaleStartDate:   "2026-02-17",
		SaleEndDate:     "2026-02-28",
		OperatorCode:    "FACO",
		Attributes: map[string]string{
			"business_unit":  "MEN",
			"color_base":     "Blue",
			"size":           "L",
			"stock":          "5",
			"status":         "active",
			"Color":          "Navy",
			"condition_type": "New",
			"material":       "Poliester",
		},
	}
}

// TestBuildProductRequestJSON verifies JSON product payload generation behavior.
func TestBuildProductRequestJSON(t *testing.T) {
	payload, err := buildProductRequestJSON(syncProductRequestFixture())
	if err != nil {
		t.Fatalf("buildProductRequestJSON() error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	request, ok := decoded["Request"].(map[string]any)
	if !ok {
		t.Fatalf("Request field missing")
	}
	product, ok := request["Product"].(map[string]any)
	if !ok {
		t.Fatalf("Product field missing")
	}
	if product["SellerSku"] != "SKU-1" {
		t.Fatalf("SellerSku = %v, want SKU-1", product["SellerSku"])
	}
	businessUnits, ok := product["BusinessUnits"].(map[string]any)
	if !ok {
		t.Fatalf("BusinessUnits = %T, want object", product["BusinessUnits"])
	}
	businessUnit, ok := businessUnits["BusinessUnit"].(map[string]any)
	if !ok {
		t.Fatalf("BusinessUnits.BusinessUnit = %T, want object", businessUnits["BusinessUnit"])
	}
	if businessUnit["OperatorCode"] != "FACO" {
		t.Fatalf("OperatorCode = %v, want FACO", businessUnit["OperatorCode"])
	}
	if businessUnit["Price"] != "100000" {
		t.Fatalf("Price = %v, want 100000", businessUnit["Price"])
	}
	if product["Color"] != "Navy" {
		t.Fatalf("Color = %v, want Navy", product["Color"])
	}
	if product["Talla"] != "L" {
		t.Fatalf("Talla = %v, want L", product["Talla"])
	}
	if product["ColorBasico"] != "Blue" {
		t.Fatalf("ColorBasico = %v, want Blue", product["ColorBasico"])
	}
}

// TestBuildProductRequestXML verifies product payload generation behavior.
func TestBuildProductRequestXML(t *testing.T) {
	payload, err := buildProductRequestXML(syncProductRequestFixture())
	if err != nil {
		t.Fatalf("buildProductRequestXML() error = %v", err)
	}

	text := string(payload)
	expected := []string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		"<Request>",
		"<Product>",
		"<SellerSku>SKU-1</SellerSku>",
		"<ParentSku>SKU-PARENT</ParentSku>",
		"<Variation>parent</Variation>",
		"<Name>Backpack Name</Name>",
		"<PrimaryCategory>1638</PrimaryCategory>",
		"<BusinessUnits>",
		"<BusinessUnit>",
		"<OperatorCode>FACO</OperatorCode>",
		"<Price>100000</Price>",
		"<Stock>5</Stock>",
		"<Status>active</Status>",
		"</BusinessUnit>",
		"</BusinessUnits>",
		"<Color>Navy</Color>",
		"<Talla>L</Talla>",
		"<ColorBasico>Blue</ColorBasico>",
		"<ProductData>",
		"<Color>Navy</Color>",
		"<Talla>L</Talla>",
		"<ColorBasico>Blue</ColorBasico>",
		"<Stock>5</Stock>",
		"<Status>active</Status>",
		"<condition_type>New</condition_type>",
		"<material>Poliester</material>",
		"</ProductData>",
		"</Product>",
		"</Request>",
	}
	for _, item := range expected {
		if !strings.Contains(text, item) {
			t.Fatalf("payload missing %q: %s", item, text)
		}
	}
}

// TestCanonicalizeProductAttributes verifies canonicalization behavior for required Falabella attributes.
func TestCanonicalizeProductAttributes(t *testing.T) {
	attributes := canonicalizeProductAttributes(map[string]string{
		"business_unit": "KIDS",
		"colorbase":     "Red",
		"size":          "M",
	})

	if attributes["BusinessUnits"] != "KIDS" {
		t.Fatalf("BusinessUnits = %q, want %q", attributes["BusinessUnits"], "KIDS")
	}
	if attributes["Color"] != "Red" {
		t.Fatalf("Color = %q, want %q", attributes["Color"], "Red")
	}
	if attributes["ColorBasico"] != "Red" {
		t.Fatalf("ColorBasico = %q, want %q", attributes["ColorBasico"], "Red")
	}
	if attributes["Talla"] != "M" {
		t.Fatalf("Talla = %q, want %q", attributes["Talla"], "M")
	}
}

// TestSanitizeXMLName verifies XML-name normalization behavior.
func TestSanitizeXMLName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: "Field"},
		{name: "snake_case", input: "condition_type", want: "condition_type"},
		{name: "invalid_chars", input: "foo bar", want: "foo_bar"},
		{name: "invalid_prefix", input: "1abc", want: "_1abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if value := sanitizeXMLName(tt.input); value != tt.want {
				t.Fatalf("sanitizeXMLName(%q) = %q, want %q", tt.input, value, tt.want)
			}
		})
	}
}
