package domain

import (
	"errors"
	"testing"
)

// TestContactValidateRequiresEmail verifies email requirement validation.
func TestContactValidateRequiresEmail(t *testing.T) {
	err := (Contact{LegalName: "Acme"}).Validate()
	if !errors.Is(err, ErrEmailRequired) {
		t.Fatalf("Validate() error = %v, want ErrEmailRequired", err)
	}
}

// TestContactValidateRejectsMixedNameStyles verifies legal/personal name exclusivity.
func TestContactValidateRejectsMixedNameStyles(t *testing.T) {
	err := (Contact{Email: "a@example.com", LegalName: "Acme", FirstName: "John", LastName: "Doe"}).Validate()
	if !errors.Is(err, ErrInvalidNameCombination) {
		t.Fatalf("Validate() error = %v, want ErrInvalidNameCombination", err)
	}
}

// TestContactValidateRequiresPersonalPair verifies both first and last names are required when legal name is absent.
func TestContactValidateRequiresPersonalPair(t *testing.T) {
	err := (Contact{Email: "a@example.com", FirstName: "John"}).Validate()
	if !errors.Is(err, ErrIncompletePersonalName) {
		t.Fatalf("Validate() error = %v, want ErrIncompletePersonalName", err)
	}
}

// TestContactValidateSuccess verifies successful validation for legal-name contacts.
func TestContactValidateSuccess(t *testing.T) {
	err := (Contact{Email: "a@example.com", LegalName: "Acme"}).Validate()
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

// TestContactNormalizeMetadata verifies metadata normalization behavior.
func TestContactNormalizeMetadata(t *testing.T) {
	entity := &Contact{
		Email:     "a@example.com",
		LegalName: "Acme",
		Metadata: map[string]string{
			" marketing.consent ": " true ",
			"":                    "ignored",
		},
	}
	entity.Normalize()

	if entity.Metadata["marketing.consent"] != "true" {
		t.Fatalf("entity.Metadata[marketing.consent] = %q, want %q", entity.Metadata["marketing.consent"], "true")
	}
	if _, exists := entity.Metadata[""]; exists {
		t.Fatalf("expected empty metadata key to be removed")
	}
}

// TestContactValidateMetadata verifies metadata validation behavior.
func TestContactValidateMetadata(t *testing.T) {
	err := (Contact{
		Email:     "a@example.com",
		LegalName: "Acme",
		Metadata:  map[string]string{"": "x"},
	}).Validate()
	if !errors.Is(err, ErrInvalidMetadata) {
		t.Fatalf("Validate() error = %v, want ErrInvalidMetadata", err)
	}
}
