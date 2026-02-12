package variation

import "testing"

// TestVariationNormalize verifies canonicalization behavior.
func TestVariationNormalize(t *testing.T) {
	entity := &Variation{Name: " Red ", Value: " #ff0000 ", Definition: Definition(" color ")}
	entity.Normalize()

	if entity.Name != "Red" {
		t.Fatalf("entity.Name = %q, want %q", entity.Name, "Red")
	}
	if entity.Value != "#ff0000" {
		t.Fatalf("entity.Value = %q, want %q", entity.Value, "#ff0000")
	}
	if entity.Definition != DefinitionColor {
		t.Fatalf("entity.Definition = %q, want %q", entity.Definition, DefinitionColor)
	}
}

// TestVariationValidate verifies variation invariants.
func TestVariationValidate(t *testing.T) {
	if err := (Variation{}).Validate(); err != ErrNameRequired {
		t.Fatalf("Validate() error = %v, want ErrNameRequired", err)
	}
	if err := (Variation{Name: "Name"}).Validate(); err != ErrValueRequired {
		t.Fatalf("Validate() error = %v, want ErrValueRequired", err)
	}
	if err := (Variation{Name: "Name", Value: "Value"}).Validate(); err != ErrDefinitionRequired {
		t.Fatalf("Validate() error = %v, want ErrDefinitionRequired", err)
	}
	if err := (Variation{Name: "Name", Value: "Value", Definition: Definition("BAD")}).Validate(); err != ErrInvalidDefinition {
		t.Fatalf("Validate() error = %v, want ErrInvalidDefinition", err)
	}

	valid := Variation{Name: "Name", Value: "Value", Definition: DefinitionText}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

// TestIsValidDefinition verifies definition helper behavior.
func TestIsValidDefinition(t *testing.T) {
	if !isValidDefinition(DefinitionColor) {
		t.Fatalf("expected DefinitionColor to be valid")
	}
	if isValidDefinition(Definition("unknown")) {
		t.Fatalf("expected unknown definition to be invalid")
	}
}
