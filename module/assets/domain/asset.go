package domain

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

var (
	// ErrIDRequired is returned when asset identifiers are missing.
	ErrIDRequired = errors.New("asset id is required")
	// ErrKeyRequired is returned when storage keys are missing.
	ErrKeyRequired = errors.New("asset key is required")
	// ErrOriginalNameRequired is returned when original file names are missing.
	ErrOriginalNameRequired = errors.New("asset originalName is required")
	// ErrMimeTypeRequired is returned when mime types are missing.
	ErrMimeTypeRequired = errors.New("asset mimeType is required")
	// ErrInvalidSize is returned when size values are not positive.
	ErrInvalidSize = errors.New("asset size must be greater than zero")
)

// Asset defines asset metadata entity values.
type Asset struct {
	// ID defines unique asset identifiers.
	ID string `json:"_id"`
	// Key defines S3 object key paths.
	Key string `json:"key"`
	// Name defines custom or derived display names.
	Name string `json:"name"`
	// OriginalName defines uploaded file names.
	OriginalName string `json:"originalName"`
	// FolderID defines optional logical folder identifiers.
	FolderID string `json:"folderId,omitempty"`
	// MimeType defines mime type values.
	MimeType string `json:"mimeType"`
	// Size defines object size in bytes.
	Size int64 `json:"size"`
	// Tags defines optional classification tags.
	Tags []Tag `json:"tags,omitempty"`
	// Metadata defines optional key-value metadata values.
	Metadata map[string]string `json:"metadata,omitempty"`
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time `json:"updatedAt"`
	// IsDeleted reports soft-delete state.
	IsDeleted bool `json:"isDeleted"`
	// DeletedAt defines soft-delete timestamps.
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// Normalize canonicalizes asset values before validation and persistence.
func (a *Asset) Normalize() {
	if a == nil {
		return
	}

	a.ID = strings.TrimSpace(a.ID)
	a.Key = strings.TrimSpace(a.Key)
	a.Name = strings.TrimSpace(a.Name)
	a.OriginalName = strings.TrimSpace(a.OriginalName)
	a.FolderID = strings.TrimSpace(a.FolderID)
	a.MimeType = strings.TrimSpace(a.MimeType)
	normalizeTags(a.Tags)
	a.Metadata = normalizeMetadata(a.Metadata)
}

// ValidateCreate verifies asset metadata required for creation.
func (a Asset) ValidateCreate() error {
	if strings.TrimSpace(a.Key) == "" {
		return ErrKeyRequired
	}
	if strings.TrimSpace(a.OriginalName) == "" {
		return ErrOriginalNameRequired
	}
	if strings.TrimSpace(a.MimeType) == "" {
		return ErrMimeTypeRequired
	}
	if a.Size <= 0 {
		return ErrInvalidSize
	}
	if err := validateTags(a.Tags); err != nil {
		return err
	}
	if err := validateMetadata(a.Metadata); err != nil {
		return err
	}

	return nil
}

// ValidateID verifies non-empty asset identifiers.
func ValidateID(id string) error {
	if strings.TrimSpace(id) == "" {
		return ErrIDRequired
	}

	return nil
}

// normalizeTags normalizes tag names and colors in-place.
func normalizeTags(tags []Tag) {
	for index := range tags {
		tags[index].Name = strings.ToLower(strings.TrimSpace(tags[index].Name))
		tags[index].Color = strings.ToLower(strings.TrimSpace(tags[index].Color))
	}
}

// normalizeMetadata normalizes metadata keys and values.
func normalizeMetadata(metadata map[string]string) map[string]string {
	if metadata == nil {
		return map[string]string{}
	}

	normalized := make(map[string]string, len(metadata))
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		normalized[trimmedKey] = strings.TrimSpace(metadata[key])
	}

	return normalized
}

// validateMetadata verifies metadata key/value constraints.
func validateMetadata(metadata map[string]string) error {
	for key, value := range metadata {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			return ErrInvalidMetadata
		}
		if len(trimmedKey) > 128 {
			return fmt.Errorf("%w: key %q exceeds max length", ErrInvalidMetadata, trimmedKey)
		}
		if len(value) > 2048 {
			return fmt.Errorf("%w: value for %q exceeds max length", ErrInvalidMetadata, trimmedKey)
		}
	}

	return nil
}
