package port

import (
	"context"
	"time"

	contactsdomain "mannaiah/module/contacts/domain"
)

// ContactSyncCommand defines normalized contact upsert payload values.
type ContactSyncCommand struct {
	// ShopDomain defines the source Shopify store domain.
	ShopDomain string
	// ShopifyID defines the source Shopify customer identifier.
	ShopifyID string
	// Email defines customer email values.
	Email string
	// DocumentType defines normalized document-type values.
	DocumentType contactsdomain.DocumentType
	// DocumentNumber defines document number values.
	DocumentNumber string
	// LegalName defines legal-name values.
	LegalName string
	// FirstName defines first-name values.
	FirstName string
	// LastName defines last-name values.
	LastName string
	// Phone defines phone values.
	Phone string
	// Address defines address line 1 values.
	Address string
	// AddressExtra defines address line 2 values.
	AddressExtra string
	// CityCode defines normalized city-code values.
	CityCode string
	// Metadata defines normalized metadata values.
	Metadata map[string]string
	// CreatedAt defines optional source creation timestamps.
	CreatedAt *time.Time
}

// ContactSyncTarget defines mainstream contact upsert behavior.
type ContactSyncTarget interface {
	// UpsertContact creates or updates one mainstream contact from Shopify values.
	UpsertContact(ctx context.Context, command ContactSyncCommand) (*contactsdomain.Contact, error)
}
