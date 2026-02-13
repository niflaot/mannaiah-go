package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	// MaxTags defines the maximum number of tags allowed on assets/folders.
	MaxTags = 5
)

var (
	// ErrTooManyTags is returned when more than MaxTags are provided.
	ErrTooManyTags = errors.New("too many tags")
	// ErrInvalidTagName is returned when tag names are empty or malformed.
	ErrInvalidTagName = errors.New("invalid tag name")
	// ErrInvalidTagColor is returned when tag colors are not lowercase hex values.
	ErrInvalidTagColor = errors.New("invalid tag color")
	// ErrDuplicatedTags is returned when tag names repeat.
	ErrDuplicatedTags = errors.New("duplicated tags")
	// ErrInvalidMetadata is returned when metadata contains invalid keys/values.
	ErrInvalidMetadata = errors.New("invalid metadata")
)

var (
	tagNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)
	tagHexPattern  = regexp.MustCompile(`^#[0-9a-f]{6}$`)
)

// Tag defines label metadata with lowercase names and lowercase hex colors.
type Tag struct {
	// Name defines lowercase tag labels.
	Name string `json:"name"`
	// Color defines lowercase hex color values.
	Color string `json:"color"`
}

// validateTags verifies tag constraints for assets/folders.
func validateTags(tags []Tag) error {
	if len(tags) > MaxTags {
		return ErrTooManyTags
	}

	seen := make(map[string]struct{}, len(tags))
	for _, value := range tags {
		name := strings.TrimSpace(value.Name)
		if !tagNamePattern.MatchString(name) {
			return fmt.Errorf("%w: %q", ErrInvalidTagName, value.Name)
		}
		if _, exists := seen[name]; exists {
			return fmt.Errorf("%w: %q", ErrDuplicatedTags, name)
		}
		seen[name] = struct{}{}

		color := strings.TrimSpace(value.Color)
		if !tagHexPattern.MatchString(color) {
			return fmt.Errorf("%w: %q", ErrInvalidTagColor, value.Color)
		}
	}

	return nil
}

// ValidateTagsForStore validates tag constraints for persistence adapters.
func ValidateTagsForStore(tags []Tag) error {
	return validateTags(tags)
}
