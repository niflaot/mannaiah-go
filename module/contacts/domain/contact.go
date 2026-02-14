package domain

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

var (
	// ErrEmailRequired is returned when contact email is missing.
	ErrEmailRequired = errors.New("contact email is required")
	// ErrInvalidNameCombination is returned when legal and personal names are mixed.
	ErrInvalidNameCombination = errors.New("cannot combine legalName with firstName/lastName")
	// ErrIncompletePersonalName is returned when personal names are partially defined.
	ErrIncompletePersonalName = errors.New("must provide either legalName or both firstName and lastName")
	// ErrInvalidMetadata is returned when metadata keys/values are invalid.
	ErrInvalidMetadata = errors.New("contact metadata is invalid")
)

// DocumentType defines supported contact document categories.
type DocumentType string

const (
	// DocumentTypeCC defines Colombian citizenship card documents.
	DocumentTypeCC DocumentType = "CC"
	// DocumentTypeCE defines foreigner id documents.
	DocumentTypeCE DocumentType = "CE"
	// DocumentTypeTI defines identity card documents.
	DocumentTypeTI DocumentType = "TI"
	// DocumentTypePAS defines passport documents.
	DocumentTypePAS DocumentType = "PAS"
	// DocumentTypeNIT defines tax id documents.
	DocumentTypeNIT DocumentType = "NIT"
	// DocumentTypeOther defines fallback document categories.
	DocumentTypeOther DocumentType = "OTHER"
)

// Contact defines the domain entity for contact management.
type Contact struct {
	// ID is the unique contact identifier.
	ID string `json:"id"`
	// DocumentType defines the document category.
	DocumentType DocumentType `json:"documentType"`
	// DocumentNumber defines the document number.
	DocumentNumber string `json:"documentNumber"`
	// LegalName defines organization legal names.
	LegalName string `json:"legalName"`
	// FirstName defines personal first names.
	FirstName string `json:"firstName"`
	// LastName defines personal last names.
	LastName string `json:"lastName"`
	// Email defines the contact email.
	Email string `json:"email"`
	// Phone defines contact phone numbers.
	Phone string `json:"phone"`
	// Address defines physical address values.
	Address string `json:"address"`
	// AddressExtra defines extra address details.
	AddressExtra string `json:"addressExtra"`
	// CityCode defines city code values.
	CityCode string `json:"cityCode"`
	// Metadata defines optional contact metadata values.
	Metadata map[string]string `json:"metadata,omitempty"`
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time `json:"updatedAt"`
}

// Normalize canonicalizes contact values before validation and persistence.
func (c *Contact) Normalize() {
	if c == nil {
		return
	}

	c.ID = strings.TrimSpace(c.ID)
	c.DocumentType = DocumentType(strings.TrimSpace(string(c.DocumentType)))
	c.DocumentNumber = strings.TrimSpace(c.DocumentNumber)
	c.LegalName = strings.TrimSpace(c.LegalName)
	c.FirstName = strings.TrimSpace(c.FirstName)
	c.LastName = strings.TrimSpace(c.LastName)
	c.Email = strings.TrimSpace(c.Email)
	c.Phone = strings.TrimSpace(c.Phone)
	c.Address = strings.TrimSpace(c.Address)
	c.AddressExtra = strings.TrimSpace(c.AddressExtra)
	c.CityCode = strings.TrimSpace(c.CityCode)
	c.Metadata = normalizeMetadata(c.Metadata)
}

// Validate verifies domain invariants for contacts.
func (c Contact) Validate() error {
	if strings.TrimSpace(c.Email) == "" {
		return ErrEmailRequired
	}

	hasLegalName := strings.TrimSpace(c.LegalName) != ""
	hasFirstName := strings.TrimSpace(c.FirstName) != ""
	hasLastName := strings.TrimSpace(c.LastName) != ""

	if hasLegalName && (hasFirstName || hasLastName) {
		return ErrInvalidNameCombination
	}
	if !hasLegalName && (!hasFirstName || !hasLastName) {
		return ErrIncompletePersonalName
	}
	if err := validateMetadata(c.Metadata); err != nil {
		return err
	}

	return nil
}

// normalizeMetadata canonicalizes metadata keys and values.
func normalizeMetadata(metadata map[string]string) map[string]string {
	if metadata == nil {
		return map[string]string{}
	}

	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	normalized := make(map[string]string, len(keys))
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
			return fmt.Errorf("%w: key exceeds max length", ErrInvalidMetadata)
		}
		if len(value) > 2048 {
			return fmt.Errorf("%w: value exceeds max length", ErrInvalidMetadata)
		}
	}

	return nil
}
