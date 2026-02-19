package service

import "testing"

// TestNormalizeScopedAttributeKey verifies canonical Falabella attribute key resolution behavior.
func TestNormalizeScopedAttributeKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "color", input: "color", want: "Color"},
		{name: "Color uppercase", input: "Color", want: "Color"},
		{name: "COLOR all-caps", input: "COLOR", want: "Color"},
		{name: "colorbase", input: "colorbase", want: "ColorBasico"},
		{name: "colorbasico", input: "colorbasico", want: "ColorBasico"},
		{name: "colorbasic", input: "colorbasic", want: "ColorBasico"},
		{name: "basiccolor", input: "basiccolor", want: "ColorBasico"},
		{name: "basecolor", input: "basecolor", want: "ColorBasico"},
		{name: "ColorBasico mixed case", input: "ColorBasico", want: "ColorBasico"},
		{name: "size", input: "size", want: "Talla"},
		{name: "talla", input: "talla", want: "Talla"},
		{name: "Talla mixed case", input: "Talla", want: "Talla"},
		{name: "businessunits", input: "businessunits", want: "BusinessUnits"},
		{name: "businessunit", input: "businessunit", want: "BusinessUnits"},
		{name: "unknown passthrough", input: "custom_field", want: "custom_field"},
		{name: "empty", input: "", want: ""},
		{name: "whitespace only", input: "   ", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeScopedAttributeKey(tt.input)
			if got != tt.want {
				t.Fatalf("normalizeScopedAttributeKey(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestNormalizeAttributeToken verifies case-insensitive alphanumeric token normalization behavior.
func TestNormalizeAttributeToken(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "simple lowercase", input: "color", want: "color"},
		{name: "mixed case", input: "ColorBasico", want: "colorbasico"},
		{name: "underscores stripped", input: "color_base", want: "colorbase"},
		{name: "hyphens stripped", input: "color-base", want: "colorbase"},
		{name: "spaces stripped", input: "color base", want: "colorbase"},
		{name: "digits preserved", input: "size42", want: "size42"},
		{name: "empty", input: "", want: ""},
		{name: "whitespace only", input: "   ", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeAttributeToken(tt.input)
			if got != tt.want {
				t.Fatalf("normalizeAttributeToken(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestNormalizeFalabellaAttributeKeys verifies top-level attribute alias canonicalization behavior.
func TestNormalizeFalabellaAttributeKeys(t *testing.T) {
	attrs := map[string]string{
		"colorbase":     "Blue",
		"size":          "L",
		"businessunits": "MEN",
		"material":      "Polyester",
	}

	normalizeFalabellaAttributeKeys(attrs)

	if attrs["ColorBasico"] != "Blue" {
		t.Fatalf("ColorBasico = %q, want %q", attrs["ColorBasico"], "Blue")
	}
	if attrs["Talla"] != "L" {
		t.Fatalf("Talla = %q, want %q", attrs["Talla"], "L")
	}
	if attrs["BusinessUnits"] != "MEN" {
		t.Fatalf("BusinessUnits = %q, want %q", attrs["BusinessUnits"], "MEN")
	}
	if attrs["material"] != "Polyester" {
		t.Fatalf("material = %q, want %q", attrs["material"], "Polyester")
	}
	if _, ok := attrs["colorbase"]; ok {
		t.Fatalf("attrs should not contain alias colorbase after normalization")
	}
	if _, ok := attrs["size"]; ok {
		t.Fatalf("attrs should not contain alias size after normalization")
	}
	if _, ok := attrs["businessunits"]; ok {
		t.Fatalf("attrs should not contain alias businessunits after normalization")
	}
}

// TestNormalizeFalabellaAttributeKeysPreservesExisting verifies existing canonical keys are not overwritten.
func TestNormalizeFalabellaAttributeKeysPreservesExisting(t *testing.T) {
	attrs := map[string]string{
		"Color":     "Navy",
		"color":     "Blue",
		"Talla":     "M",
		"size":      "L",
		"colorbase": "Red",
	}

	normalizeFalabellaAttributeKeys(attrs)

	if attrs["Color"] != "Navy" {
		t.Fatalf("Color = %q, want %q (should preserve existing)", attrs["Color"], "Navy")
	}
	if attrs["Talla"] != "M" {
		t.Fatalf("Talla = %q, want %q (should preserve existing)", attrs["Talla"], "M")
	}
	if attrs["ColorBasico"] != "Red" {
		t.Fatalf("ColorBasico = %q, want %q", attrs["ColorBasico"], "Red")
	}
}

// TestNormalizeFalabellaAttributeKeysEmpty verifies empty and nil maps are handled safely.
func TestNormalizeFalabellaAttributeKeysEmpty(t *testing.T) {
	normalizeFalabellaAttributeKeys(nil)
	normalizeFalabellaAttributeKeys(map[string]string{})
}

// TestSplitScopedAttributeKey verifies scoped key splitting behavior.
func TestSplitScopedAttributeKey(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		wantScope string
		wantField string
		wantOK    bool
	}{
		{name: "valid scoped", key: "0013.color", wantScope: "0013", wantField: "color", wantOK: true},
		{name: "valid multi-dot", key: "SKU.color.extra", wantScope: "SKU", wantField: "color.extra", wantOK: true},
		{name: "no dot", key: "material", wantOK: false},
		{name: "empty scope", key: ".color", wantOK: false},
		{name: "empty field", key: "0013.", wantOK: false},
		{name: "empty input", key: "", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scope, field, ok := splitScopedAttributeKey(tt.key)
			if ok != tt.wantOK {
				t.Fatalf("splitScopedAttributeKey(%q) ok = %v, want %v", tt.key, ok, tt.wantOK)
			}
			if ok {
				if scope != tt.wantScope {
					t.Fatalf("scope = %q, want %q", scope, tt.wantScope)
				}
				if field != tt.wantField {
					t.Fatalf("field = %q, want %q", field, tt.wantField)
				}
			}
		})
	}
}

// TestNormalizeScopedVariantToken verifies case-insensitive variant token normalization behavior.
func TestNormalizeScopedVariantToken(t *testing.T) {
	if got := normalizeScopedVariantToken("SKU-001"); got != "sku-001" {
		t.Fatalf("normalizeScopedVariantToken(%q) = %q, want %q", "SKU-001", got, "sku-001")
	}
	if got := normalizeScopedVariantToken("  "); got != "" {
		t.Fatalf("normalizeScopedVariantToken(whitespace) = %q, want empty", got)
	}
}

// TestApplyVariantScopedAttributesNilTarget verifies nil-target safety.
func TestApplyVariantScopedAttributesNilTarget(t *testing.T) {
	applyVariantScopedAttributes(nil, map[string]string{"0013.color": "Navy"}, "0013", map[string]struct{}{"0013": {}})
}

// TestApplyVariantScopedAttributesEmptySource verifies no-op on empty source.
func TestApplyVariantScopedAttributesEmptySource(t *testing.T) {
	target := map[string]string{"material": "Polyester"}
	applyVariantScopedAttributes(target, nil, "0013", map[string]struct{}{"0013": {}})
	if target["material"] != "Polyester" {
		t.Fatalf("material = %q, want %q", target["material"], "Polyester")
	}
}

// TestApplyVariantScopedAttributesEmptyVariantSKU verifies no-op on empty variant SKU.
func TestApplyVariantScopedAttributesEmptyVariantSKU(t *testing.T) {
	target := map[string]string{"material": "Polyester"}
	applyVariantScopedAttributes(target, map[string]string{"0013.color": "Navy"}, "  ", map[string]struct{}{"0013": {}})
	if target["material"] != "Polyester" {
		t.Fatalf("material = %q, want %q", target["material"], "Polyester")
	}
}

// TestNormalizeKnownVariantSKUs verifies known-SKU normalization with fallback behavior.
func TestNormalizeKnownVariantSKUs(t *testing.T) {
	known := normalizeKnownVariantSKUs(map[string]struct{}{"SKU-A": {}, "SKU-B": {}}, "default")
	if _, ok := known["sku-a"]; !ok {
		t.Fatalf("expected normalized sku-a")
	}
	if _, ok := known["sku-b"]; !ok {
		t.Fatalf("expected normalized sku-b")
	}

	fallback := normalizeKnownVariantSKUs(nil, "default-sku")
	if _, ok := fallback["default-sku"]; !ok {
		t.Fatalf("expected fallback sku")
	}

	emptyKnown := normalizeKnownVariantSKUs(map[string]struct{}{"  ": {}}, "fallback")
	if _, ok := emptyKnown["fallback"]; !ok {
		t.Fatalf("expected fallback when all known SKUs are whitespace-only")
	}
}
