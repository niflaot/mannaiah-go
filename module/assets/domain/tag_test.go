package domain

import "testing"

// TestValidateTags verifies tag normalization constraints.
func TestValidateTags(t *testing.T) {
	if err := validateTags([]Tag{{Name: "UPPER", Color: "#ffffff"}}); err == nil {
		t.Fatalf("expected invalid uppercase tag name")
	}
	if err := validateTags([]Tag{{Name: "valid", Color: "#ABCDEF"}}); err == nil {
		t.Fatalf("expected invalid uppercase tag color")
	}
	if err := validateTags([]Tag{{Name: "valid", Color: "#ff00"}}); err == nil {
		t.Fatalf("expected invalid short tag color")
	}
	if err := validateTags([]Tag{{Name: "dup", Color: "#ffffff"}, {Name: "dup", Color: "#000000"}}); err == nil {
		t.Fatalf("expected duplicated tag error")
	}
	if err := validateTags([]Tag{
		{Name: "a", Color: "#111111"},
		{Name: "b", Color: "#222222"},
		{Name: "c", Color: "#333333"},
		{Name: "d", Color: "#444444"},
		{Name: "e", Color: "#555555"},
		{Name: "f", Color: "#666666"},
	}); err == nil {
		t.Fatalf("expected too many tags error")
	}
	if err := validateTags([]Tag{{Name: "valid-tag", Color: "#abcdef"}}); err != nil {
		t.Fatalf("validateTags(valid) error = %v", err)
	}
}
