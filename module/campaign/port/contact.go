package port

import (
	"context"
	"time"
)

// ContactData defines per-contact data used to populate campaign template contexts.
type ContactData struct {
	// Name is the contact display name.
	Name string
	// Email is the contact email address.
	Email string
	// LastSaleDate is the date of the contact's most recent purchase, or nil if unknown.
	LastSaleDate *time.Time
}

// ContactDataProvider defines per-contact personalization data fetch behavior.
type ContactDataProvider interface {
	// GetContactData returns personalization data for one contact.
	// Returns a zero-value ContactData (not an error) when the contact has no sales history.
	GetContactData(ctx context.Context, contactID string) (ContactData, error)
}

// NoopContactDataProvider returns zero-value ContactData for all contacts.
type NoopContactDataProvider struct{}

// GetContactData returns an empty ContactData.
func (NoopContactDataProvider) GetContactData(_ context.Context, _ string) (ContactData, error) {
	return ContactData{}, nil
}
