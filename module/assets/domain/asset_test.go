package domain

import "testing"

// TestNormalize verifies asset normalization behavior.
func TestNormalize(t *testing.T) {
	asset := &Asset{ID: " a-1 ", Key: " assets/k ", Name: " Name ", OriginalName: " file.png ", MimeType: " image/png "}
	asset.Normalize()

	if asset.ID != "a-1" {
		t.Fatalf("asset.ID = %q, want %q", asset.ID, "a-1")
	}
	if asset.Key != "assets/k" {
		t.Fatalf("asset.Key = %q, want %q", asset.Key, "assets/k")
	}
	if asset.Name != "Name" {
		t.Fatalf("asset.Name = %q, want %q", asset.Name, "Name")
	}
}

// TestValidateCreate verifies create validation behavior.
func TestValidateCreate(t *testing.T) {
	if err := (Asset{}).ValidateCreate(); err != ErrKeyRequired {
		t.Fatalf("ValidateCreate() error = %v, want ErrKeyRequired", err)
	}
	if err := (Asset{Key: "k"}).ValidateCreate(); err != ErrOriginalNameRequired {
		t.Fatalf("ValidateCreate() error = %v, want ErrOriginalNameRequired", err)
	}
	if err := (Asset{Key: "k", OriginalName: "o"}).ValidateCreate(); err != ErrMimeTypeRequired {
		t.Fatalf("ValidateCreate() error = %v, want ErrMimeTypeRequired", err)
	}
	if err := (Asset{Key: "k", OriginalName: "o", MimeType: "m"}).ValidateCreate(); err != ErrInvalidSize {
		t.Fatalf("ValidateCreate() error = %v, want ErrInvalidSize", err)
	}

	valid := Asset{Key: "k", OriginalName: "o", MimeType: "m", Size: 1}
	if err := valid.ValidateCreate(); err != nil {
		t.Fatalf("ValidateCreate() error = %v, want nil", err)
	}
}

// TestValidateID verifies id validation behavior.
func TestValidateID(t *testing.T) {
	if err := ValidateID(""); err != ErrIDRequired {
		t.Fatalf("ValidateID() error = %v, want ErrIDRequired", err)
	}
	if err := ValidateID("a-1"); err != nil {
		t.Fatalf("ValidateID() error = %v, want nil", err)
	}
}
