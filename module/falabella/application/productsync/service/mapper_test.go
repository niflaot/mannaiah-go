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
						"Brand":                  "GENÉRICO",
						"Model":                  "M-1",
						"TaxClass":               "IVA 19%",
						"PriceFalabella":         "100000",
						"QuantityFalabella":      "12",
						"SalePriceFalabella":     "90000",
						"SaleStartDateFalabella": "2026-02-19 00:00:00",
						"SaleEndDateFalabella":   "2026-02-20 23:59:59",
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
	if request.Price != "100000" {
		t.Fatalf("Price = %q, want %q", request.Price, "100000")
	}
	if request.SalePrice != "90000" {
		t.Fatalf("SalePrice = %q, want %q", request.SalePrice, "90000")
	}
	if request.TaxClass != "IVA 19%" {
		t.Fatalf("TaxClass = %q, want %q", request.TaxClass, "IVA 19%")
	}
	if request.Brand != "GENÉRICO" {
		t.Fatalf("Brand = %q, want %q", request.Brand, "GENÉRICO")
	}
	if request.Attributes["Stock"] != "12" {
		t.Fatalf("Stock = %q, want %q", request.Attributes["Stock"], "12")
	}
	if _, ok := request.Attributes["Brand"]; ok {
		t.Fatalf("Brand should not be present inside ProductData attributes")
	}
	if _, ok := request.Attributes["brand"]; ok {
		t.Fatalf("brand alias should not be present inside ProductData attributes")
	}
	if _, ok := request.Attributes["global_identifier"]; ok {
		t.Fatalf("global_identifier should not be injected into falabella attributes")
	}
}

// TestMapProductCanonicalizesKnownAttributeAliases verifies known Falabella attribute key canonicalization behavior.
func TestMapProductCanonicalizesKnownAttributeAliases(t *testing.T) {
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
						"businessunits": "1",
						"colorbase":     "Blue",
						"size":          "L",
						"color":         "Navy",
					},
				},
			},
		},
		Config{Realm: "falabella"},
	)
	if err != nil {
		t.Fatalf("mapProduct() error = %v", err)
	}
	if skipReason != "" {
		t.Fatalf("skipReason = %q, want empty", skipReason)
	}
	if request.Attributes["BusinessUnits"] != "1" {
		t.Fatalf("BusinessUnits = %q, want %q", request.Attributes["BusinessUnits"], "1")
	}
	if request.Attributes["ColorBasico"] != "Blue" {
		t.Fatalf("ColorBasico = %q, want %q", request.Attributes["ColorBasico"], "Blue")
	}
	if request.Attributes["Talla"] != "L" {
		t.Fatalf("Talla = %q, want %q", request.Attributes["Talla"], "L")
	}
	if request.Attributes["Color"] != "Navy" {
		t.Fatalf("Color = %q, want %q", request.Attributes["Color"], "Navy")
	}
	if _, ok := request.Attributes["businessunits"]; ok {
		t.Fatalf("request.Attributes should not contain businessunits alias after canonicalization")
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
	if _, _, err := mapProduct(
		port.CatalogProduct{SKU: "SKU-1", Datasheets: []port.CatalogDatasheet{{Realm: "falabella", Name: "Backpack"}}},
		Config{Realm: "falabella"},
	); !errorspkg.Is(err, ErrDescriptionRequired) {
		t.Fatalf("mapProduct(empty-description) error = %v, want ErrDescriptionRequired", err)
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

	unique := uniqueTrimmedValues([]string{" a ", "a", "", "b"})
	if len(unique) != 2 || unique[0] != "a" || unique[1] != "b" {
		t.Fatalf("uniqueTrimmedValues() = %#v, want [a b]", unique)
	}
}

// TestMapVariantProduct verifies variant request mapping behavior.
func TestMapVariantProduct(t *testing.T) {
	base := port.SyncProductRequest{
		SKU:   "PARENT-SKU",
		Name:  "Backpack",
		Brand: "GENERIC",
		Attributes: map[string]string{
			"material": "Polyester",
		},
	}

	request, err := mapVariantProduct(base, port.CatalogVariant{
		SKU: "CHILD-SKU",
		Variations: []port.CatalogVariation{
			{Definition: "COLOR", Value: "Blue"},
			{Definition: "SIZE", Value: "L"},
			{Name: "Material", Definition: "TEXT", Value: "Canvas"},
		},
	}, map[string]struct{}{"child-sku": {}})
	if err != nil {
		t.Fatalf("mapVariantProduct() error = %v", err)
	}

	if request.SKU != "CHILD-SKU" {
		t.Fatalf("request.SKU = %q, want %q", request.SKU, "CHILD-SKU")
	}
	if request.ParentSKU != "PARENT-SKU" {
		t.Fatalf("request.ParentSKU = %q, want %q", request.ParentSKU, "PARENT-SKU")
	}
	if request.Attributes["Color"] != "Blue" {
		t.Fatalf("request.Attributes[Color] = %q, want %q", request.Attributes["Color"], "Blue")
	}
	if request.Attributes["ColorBasico"] != "Blue" {
		t.Fatalf("request.Attributes[ColorBasico] = %q, want %q", request.Attributes["ColorBasico"], "Blue")
	}
	if request.Attributes["Talla"] != "L" {
		t.Fatalf("request.Attributes[Talla] = %q, want %q", request.Attributes["Talla"], "L")
	}
	if request.Attributes["Material"] != "Canvas" {
		t.Fatalf("request.Attributes[Material] = %q, want %q", request.Attributes["Material"], "Canvas")
	}
}

// TestMapVariantProductValidation verifies variant mapping validation behavior.
func TestMapVariantProductValidation(t *testing.T) {
	if _, err := mapVariantProduct(port.SyncProductRequest{SKU: "PARENT-SKU"}, port.CatalogVariant{SKU: " "}, map[string]struct{}{"child-sku": {}}); !errorspkg.Is(err, ErrVariantSKURequired) {
		t.Fatalf("mapVariantProduct(empty-sku) error = %v, want ErrVariantSKURequired", err)
	}
}

// TestMapVariantProductScopedAttributes verifies variant-SKU scoped attribute mapping behavior.
func TestMapVariantProductScopedAttributes(t *testing.T) {
	base := port.SyncProductRequest{
		SKU: "PARENT-SKU",
		Attributes: map[string]string{
			"material":                    "Polyester",
			"material.type":               "Canvas",
			"not_a_variant.color":         "Green",
			"0013.color":                  "Navy",
			"0013.colorbase":              "Blue",
			"0013.business_units":         "MEN",
			"0013.custom_attribute":       "custom-0013",
			"0013.size":                   "L",
			"0013.tax_percentage":         "19",
			"0013.PriceFalabella":         "165000",
			"0013.QuantityFalabella":      "9",
			"0013.SalePriceFalabella":     "159000",
			"0013.SaleStartDateFalabella": "2026-02-19 00:00:00",
			"0013.SaleEndDateFalabella":   "2026-02-20 23:59:59",
			"0014.color":                  "Black",
			"0014.custom_attribute":       "custom-0014",
			"0014.PriceFalabella":         "170000",
			"0014.QuantityFalabella":      "5",
		},
	}

	request0013, err := mapVariantProduct(base, port.CatalogVariant{
		SKU: "0013",
		Variations: []port.CatalogVariation{
			{Definition: "COLOR", Value: "Red"},
			{Definition: "SIZE", Value: "M"},
		},
	}, map[string]struct{}{"0013": {}, "0014": {}})
	if err != nil {
		t.Fatalf("mapVariantProduct() error = %v", err)
	}

	if request0013.Attributes["Color"] != "Navy" {
		t.Fatalf("request0013.Attributes[Color] = %q, want %q", request0013.Attributes["Color"], "Navy")
	}
	if request0013.Attributes["ColorBasico"] != "Blue" {
		t.Fatalf("request0013.Attributes[ColorBasico] = %q, want %q", request0013.Attributes["ColorBasico"], "Blue")
	}
	if request0013.Attributes["BusinessUnits"] != "MEN" {
		t.Fatalf("request0013.Attributes[BusinessUnits] = %q, want %q", request0013.Attributes["BusinessUnits"], "MEN")
	}
	if request0013.Attributes["Talla"] != "L" {
		t.Fatalf("request0013.Attributes[Talla] = %q, want %q", request0013.Attributes["Talla"], "L")
	}
	if request0013.Attributes["custom_attribute"] != "custom-0013" {
		t.Fatalf("request0013.Attributes[custom_attribute] = %q, want %q", request0013.Attributes["custom_attribute"], "custom-0013")
	}
	if request0013.Attributes["TaxPercentage"] != "19" {
		t.Fatalf("request0013.Attributes[TaxPercentage] = %q, want %q", request0013.Attributes["TaxPercentage"], "19")
	}
	if request0013.Price != "165000" {
		t.Fatalf("request0013.Price = %q, want %q", request0013.Price, "165000")
	}
	if request0013.SalePrice != "159000" {
		t.Fatalf("request0013.SalePrice = %q, want %q", request0013.SalePrice, "159000")
	}
	if request0013.SaleStartDate != "2026-02-19 00:00:00" {
		t.Fatalf("request0013.SaleStartDate = %q, want %q", request0013.SaleStartDate, "2026-02-19 00:00:00")
	}
	if request0013.SaleEndDate != "2026-02-20 23:59:59" {
		t.Fatalf("request0013.SaleEndDate = %q, want %q", request0013.SaleEndDate, "2026-02-20 23:59:59")
	}
	if request0013.Attributes["Stock"] != "9" {
		t.Fatalf("request0013.Attributes[Stock] = %q, want %q", request0013.Attributes["Stock"], "9")
	}
	if request0013.Attributes["material"] != "Polyester" {
		t.Fatalf("request0013.Attributes[material] = %q, want %q", request0013.Attributes["material"], "Polyester")
	}
	if request0013.Attributes["material.type"] != "Canvas" {
		t.Fatalf("request0013.Attributes[material.type] = %q, want %q", request0013.Attributes["material.type"], "Canvas")
	}
	if request0013.Attributes["not_a_variant.color"] != "Green" {
		t.Fatalf("request0013.Attributes[not_a_variant.color] = %q, want %q", request0013.Attributes["not_a_variant.color"], "Green")
	}
	if _, ok := request0013.Attributes["0013.color"]; ok {
		t.Fatalf("request0013.Attributes should not contain scoped key 0013.color")
	}
	if _, ok := request0013.Attributes["0014.color"]; ok {
		t.Fatalf("request0013.Attributes should not contain scoped key 0014.color")
	}

	request0014, err := mapVariantProduct(base, port.CatalogVariant{
		SKU: "0014",
		Variations: []port.CatalogVariation{
			{Definition: "COLOR", Value: "Black"},
		},
	}, map[string]struct{}{"0013": {}, "0014": {}})
	if err != nil {
		t.Fatalf("mapVariantProduct() error = %v", err)
	}
	if request0014.Attributes["Color"] != "Black" {
		t.Fatalf("request0014.Attributes[Color] = %q, want %q", request0014.Attributes["Color"], "Black")
	}
	if request0014.Attributes["custom_attribute"] != "custom-0014" {
		t.Fatalf("request0014.Attributes[custom_attribute] = %q, want %q", request0014.Attributes["custom_attribute"], "custom-0014")
	}
	if request0014.Price != "170000" {
		t.Fatalf("request0014.Price = %q, want %q", request0014.Price, "170000")
	}
	if request0014.Attributes["Stock"] != "5" {
		t.Fatalf("request0014.Attributes[Stock] = %q, want %q", request0014.Attributes["Stock"], "5")
	}
	if _, ok := request0014.Attributes["0013.color"]; ok {
		t.Fatalf("request0014.Attributes should not contain scoped key 0013.color")
	}
}

// TestResolveImageURLs verifies image URL filtering behavior.
func TestResolveImageURLs(t *testing.T) {
	images := []port.CatalogImage{
		{URL: "https://cdn.example.com/parent.jpg", Position: intPtr(2)},
		{URL: "https://cdn.example.com/variant.jpg", Position: intPtr(4), VariationPosition: intPtr(1), VariationIDs: []string{"v-color"}},
		{URL: "https://cdn.example.com/variant-priority.jpg", Position: intPtr(3), VariationPosition: intPtr(0), VariationIDs: []string{"v-color"}},
		{URL: "https://cdn.example.com/parent-priority.jpg", Position: intPtr(0)},
		{URL: "https://cdn.example.com/excluded.jpg", IncludedRealms: []string{"woo"}},
	}

	parentURLs := resolveImageURLs(images, "falabella", nil)
	if len(parentURLs) != 2 {
		t.Fatalf("parentURLs = %#v, want 2 urls", parentURLs)
	}
	if parentURLs[0] != "https://cdn.example.com/parent-priority.jpg" || parentURLs[1] != "https://cdn.example.com/parent.jpg" {
		t.Fatalf("parentURLs = %#v, want parent-priority then parent", parentURLs)
	}

	variantURLs := resolveImageURLs(images, "falabella", []string{"v-color", "v-size"})
	if len(variantURLs) != 4 {
		t.Fatalf("variantURLs = %#v, want 4 urls", variantURLs)
	}
	expected := []string{
		"https://cdn.example.com/variant-priority.jpg",
		"https://cdn.example.com/variant.jpg",
		"https://cdn.example.com/parent-priority.jpg",
		"https://cdn.example.com/parent.jpg",
	}
	for index, expectedURL := range expected {
		if variantURLs[index] != expectedURL {
			t.Fatalf("variantURLs[%d] = %q, want %q", index, variantURLs[index], expectedURL)
		}
	}
}

func intPtr(value int) *int {
	resolved := value
	return &resolved
}
