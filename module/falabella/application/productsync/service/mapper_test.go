package service

import (
	errorspkg "errors"
	"testing"

	"mannaiah/module/falabella/port"
)

// TestMapProduct verifies product-to-sync mapping behavior.
func TestMapProduct(t *testing.T) {
	request, skipReason, err := mapProduct(
		port.CatalogProduct{
			ID:  "p-1",
			SKU: "SKU-1",
			Datasheets: []port.CatalogDatasheet{
				{
					Realm:       "falabella",
					Name:        "Backpack",
					Description: "Desc",
					Attributes: map[string]any{
						"brand":             "GENERIC",
						"model":             "M-1",
						"tax_percentage":    "19%",
						"price_falabella":   "100000",
						"sale_price_falabella": "90000",
					},
				},
			},
		},
		Config{Realm: "falabella", CategoryID: "1638", GlobalIdentifier: "G08010305", AttributeSetID: "5"},
	)
	if err != nil {
		t.Fatalf("mapProduct() error = %v", err)
	}
	if skipReason != "" {
		t.Fatalf("skipReason = %q, want empty", skipReason)
	}
	if request.PrimaryCategory != "1638" {
		t.Fatalf("PrimaryCategory = %q, want %q", request.PrimaryCategory, "1638")
	}
	if request.Attributes["global_identifier"] != "G08010305" {
		t.Fatalf("global_identifier = %q, want %q", request.Attributes["global_identifier"], "G08010305")
	}
}

// TestMapProductSkipsMissingRealm verifies realm filtering behavior.
func TestMapProductSkipsMissingRealm(t *testing.T) {
	_, skipReason, err := mapProduct(
		port.CatalogProduct{SKU: "SKU-1", Datasheets: []port.CatalogDatasheet{{Realm: "default", Name: "Name"}}},
		Config{Realm: "falabella"},
	)
	if err != nil {
		t.Fatalf("mapProduct() error = %v", err)
	}
	if skipReason != "missing_falabella_realm" {
		t.Fatalf("skipReason = %q, want %q", skipReason, "missing_falabella_realm")
	}
}

// TestMapProductValidation verifies mapping validation behavior.
func TestMapProductValidation(t *testing.T) {
	if _, _, err := mapProduct(port.CatalogProduct{}, Config{Realm: "falabella"}); !errorspkg.Is(err, ErrSKURequired) {
		t.Fatalf("mapProduct(empty-sku) error = %v, want ErrSKURequired", err)
	}
	if _, _, err := mapProduct(
		port.CatalogProduct{SKU: "SKU-1", Datasheets: []port.CatalogDatasheet{{Realm: "falabella"}}},
		Config{Realm: "falabella"},
	); !errorspkg.Is(err, ErrNameRequired) {
		t.Fatalf("mapProduct(empty-name) error = %v, want ErrNameRequired", err)
	}
}

// TestHelpers verifies mapper helper behavior.
func TestHelpers(t *testing.T) {
	if value := firstNonEmpty("", "  ", "ok", "x"); value != "ok" {
		t.Fatalf("firstNonEmpty() = %q, want %q", value, "ok")
	}
	if value := firstNonEmpty("", "  "); value != "" {
		t.Fatalf("firstNonEmpty(empty) = %q, want empty", value)
	}

	datasheet, ok := findDatasheetByRealm([]port.CatalogDatasheet{{Realm: "Falabella", Name: "N"}}, "falabella")
	if !ok || datasheet.Name != "N" {
		t.Fatalf("findDatasheetByRealm() = (%#v,%v), want name N and true", datasheet, ok)
	}
	if _, ok := findDatasheetByRealm(nil, "falabella"); ok {
		t.Fatalf("findDatasheetByRealm(nil) should return false")
	}

	values := toStringMap(map[string]any{"a": 1, " b ": "x", "": "skip", "nil": nil})
	if values["a"] != "1" || values["b"] != "x" {
		t.Fatalf("toStringMap() = %#v, want a=1 b=x", values)
	}
	if _, ok := values["nil"]; ok {
		t.Fatalf("toStringMap() should omit nil values")
	}
}

