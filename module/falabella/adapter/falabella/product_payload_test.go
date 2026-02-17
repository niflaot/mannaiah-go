package falabella

import (
	"strings"
	"testing"

	"mannaiah/module/falabella/port"
)

// syncProductRequestFixture builds sync-product test fixtures.
func syncProductRequestFixture() port.SyncProductRequest {
	return port.SyncProductRequest{
		SKU:             "SKU-1",
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
		Attributes: map[string]string{
			"condition_type": "New",
			"material":       "Poliester",
		},
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
		"<Request>",
		"<Product>",
		"<SellerSku>SKU-1</SellerSku>",
		"<Name>Backpack Name</Name>",
		"<PrimaryCategory>1638</PrimaryCategory>",
		"<ProductData>",
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

