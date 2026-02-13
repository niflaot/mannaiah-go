package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

var (
	// ErrFolderIDRequired is returned when folder identifiers are missing.
	ErrFolderIDRequired = errors.New("asset folder id is required")
	// ErrFolderNameRequired is returned when folder names are missing.
	ErrFolderNameRequired = errors.New("asset folder name is required")
	// ErrFolderSlugInvalid is returned when folder slug generation fails.
	ErrFolderSlugInvalid = errors.New("asset folder slug is invalid")
)

var (
	folderSlugSanitizer = regexp.MustCompile(`[^a-z0-9-]`)
	folderSlugRepeated  = regexp.MustCompile(`-+`)
)

// Folder defines logical asset folder metadata.
type Folder struct {
	// ID defines unique folder identifiers.
	ID string `json:"_id"`
	// Name defines user-facing folder names.
	Name string `json:"name"`
	// Slug defines normalized folder slugs used in logical paths.
	Slug string `json:"slug"`
	// Tags defines optional classification tags.
	Tags []Tag `json:"tags,omitempty"`
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time `json:"updatedAt"`
	// IsDeleted reports soft-delete state.
	IsDeleted bool `json:"isDeleted"`
	// DeletedAt defines soft-delete timestamps.
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// Normalize canonicalizes folder values before validation and persistence.
func (f *Folder) Normalize() {
	if f == nil {
		return
	}

	f.ID = strings.TrimSpace(f.ID)
	f.Name = strings.TrimSpace(f.Name)
	f.Slug = strings.TrimSpace(strings.ToLower(f.Slug))
	if f.Slug == "" {
		f.Slug = BuildFolderSlug(f.Name)
	}
	normalizeTags(f.Tags)
}

// ValidateCreate verifies folder data required for creation.
func (f Folder) ValidateCreate() error {
	if strings.TrimSpace(f.Name) == "" {
		return ErrFolderNameRequired
	}
	if strings.TrimSpace(f.Slug) == "" {
		return ErrFolderSlugInvalid
	}
	if err := validateTags(f.Tags); err != nil {
		return err
	}

	return nil
}

// ValidateFolderID verifies non-empty folder identifiers.
func ValidateFolderID(id string) error {
	if strings.TrimSpace(id) == "" {
		return ErrFolderIDRequired
	}

	return nil
}

// BuildFolderSlug builds deterministic folder slugs from names.
func BuildFolderSlug(name string) string {
	trimmed := strings.ToLower(strings.TrimSpace(name))
	trimmed = strings.ReplaceAll(trimmed, " ", "-")
	trimmed = folderSlugSanitizer.ReplaceAllString(trimmed, "-")
	trimmed = folderSlugRepeated.ReplaceAllString(trimmed, "-")
	trimmed = strings.Trim(trimmed, "-")

	return trimmed
}
