package domain

import (
	"errors"
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
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time `json:"updatedAt"`
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

	return nil
}
