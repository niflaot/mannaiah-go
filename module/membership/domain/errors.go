package domain

import (
	"errors"
	"strings"
	"unicode"
)

var (
	// ErrInvalidContactID is returned when contact id values are invalid.
	ErrInvalidContactID = errors.New("membership contact id is required")
	// ErrInvalidEmail is returned when email values are invalid.
	ErrInvalidEmail = errors.New("membership email is required")
	// ErrInvalidChannel is returned when channel values are invalid.
	ErrInvalidChannel = errors.New("membership channel is invalid")
	// ErrInvalidAction is returned when action values are invalid.
	ErrInvalidAction = errors.New("membership action is invalid")
	// ErrStatusNotFound is returned when membership status rows are missing.
	ErrStatusNotFound = errors.New("membership status not found")
	// ErrContactNotFound is returned when source contacts are not found.
	ErrContactNotFound = errors.New("membership contact not found")
)

// IsValid reports whether the channel is recognized.
func (c Channel) IsValid() bool {
	value := strings.ToLower(strings.TrimSpace(string(c)))
	if value == "" {
		return false
	}

	for _, character := range value {
		if unicode.IsLetter(character) || unicode.IsDigit(character) || character == '_' || character == '-' {
			continue
		}
		return false
	}

	return true
}

// IsValid reports whether the action is recognized.
func (a Action) IsValid() bool {
	switch a {
	case ActionOptIn, ActionOptOut:
		return true
	default:
		return false
	}
}
