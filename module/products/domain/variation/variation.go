package variation

import (
	"errors"
	"strings"
	"time"
)

var (
	// ErrNameRequired is returned when variation names are missing.
	ErrNameRequired = errors.New("variation name is required")
	// ErrValueRequired is returned when variation values are missing.
	ErrValueRequired = errors.New("variation value is required")
	// ErrDefinitionRequired is returned when variation definitions are missing.
	ErrDefinitionRequired = errors.New("variation definition is required")
	// ErrInvalidDefinition is returned when variation definitions are invalid.
	ErrInvalidDefinition = errors.New("variation definition is invalid")
)

// Definition defines allowed variation definition values.
type Definition string

const (
	// DefinitionColor defines color variations.
	DefinitionColor Definition = "COLOR"
	// DefinitionSize defines size variations.
	DefinitionSize Definition = "SIZE"
	// DefinitionText defines generic text variations.
	DefinitionText Definition = "TEXT"
)

// Variation defines the variation domain entity.
type Variation struct {
	// ID is the unique variation identifier.
	ID string `json:"_id"`
	// Name is the human-readable variation label.
	Name string `json:"name"`
	// Definition identifies variation type.
	Definition Definition `json:"definition"`
	// Value is the machine-readable variation value.
	Value string `json:"value"`
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time `json:"updatedAt"`
	// IsDeleted reports soft-delete state.
	IsDeleted bool `json:"isDeleted"`
	// DeletedAt defines soft-delete timestamps.
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// Normalize canonicalizes variation values before persistence and validation.
func (v *Variation) Normalize() {
	if v == nil {
		return
	}

	v.Name = strings.TrimSpace(v.Name)
	v.Value = strings.TrimSpace(v.Value)
	v.Definition = Definition(strings.ToUpper(strings.TrimSpace(string(v.Definition))))
}

// Validate verifies variation domain invariants.
func (v Variation) Validate() error {
	if strings.TrimSpace(v.Name) == "" {
		return ErrNameRequired
	}
	if strings.TrimSpace(v.Value) == "" {
		return ErrValueRequired
	}

	definition := strings.TrimSpace(string(v.Definition))
	if definition == "" {
		return ErrDefinitionRequired
	}
	if !isValidDefinition(v.Definition) {
		return ErrInvalidDefinition
	}

	return nil
}

// isValidDefinition reports whether the definition enum is valid.
func isValidDefinition(definition Definition) bool {
	switch definition {
	case DefinitionColor, DefinitionSize, DefinitionText:
		return true
	default:
		return false
	}
}
