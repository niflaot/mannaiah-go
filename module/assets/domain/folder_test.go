package domain

import "testing"

// TestBuildFolderSlug verifies folder slug generation behavior.
func TestBuildFolderSlug(t *testing.T) {
	if got := BuildFolderSlug("  Hero Images / Summer  "); got != "hero-images-summer" {
		t.Fatalf("BuildFolderSlug() = %q, want %q", got, "hero-images-summer")
	}
}

// TestFolderValidation verifies folder validation behavior.
func TestFolderValidation(t *testing.T) {
	folder := &Folder{Name: "  Hero Images  ", Tags: []Tag{{Name: "hero", Color: "#ff0000"}}}
	folder.Normalize()

	if folder.Slug != "hero-images" {
		t.Fatalf("folder.Slug = %q, want %q", folder.Slug, "hero-images")
	}
	if err := folder.ValidateCreate(); err != nil {
		t.Fatalf("ValidateCreate() error = %v", err)
	}

	if err := (&Folder{}).ValidateCreate(); err != ErrFolderNameRequired {
		t.Fatalf("ValidateCreate(empty) error = %v, want ErrFolderNameRequired", err)
	}
	if err := ValidateFolderID(""); err != ErrFolderIDRequired {
		t.Fatalf("ValidateFolderID(empty) error = %v, want ErrFolderIDRequired", err)
	}
}
