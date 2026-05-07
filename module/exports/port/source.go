package port

import (
	"context"
	"time"
)

// ContactRow defines flattened contact export values.
type ContactRow struct {
	// ID defines contact identifier values.
	ID string
	// DocumentType defines document category values.
	DocumentType string
	// DocumentNumber defines document number values.
	DocumentNumber string
	// LegalName defines legal name values.
	LegalName string
	// FirstName defines first name values.
	FirstName string
	// LastName defines last name values.
	LastName string
	// Email defines email values.
	Email string
	// Phone defines phone values.
	Phone string
	// Address defines address line 1 values.
	Address string
	// AddressExtra defines address line 2 values.
	AddressExtra string
	// CityCode defines city code values.
	CityCode string
	// MembershipOptIn defines whether latest membership consent is opted in.
	MembershipOptIn bool
	// MembershipOptInAt defines latest membership consent decision timestamps.
	MembershipOptInAt time.Time
	// PrivacyAccepted defines whether privacy consent was accepted.
	PrivacyAccepted bool
	// PrivacyAcceptedAt defines privacy consent acceptance timestamps.
	PrivacyAcceptedAt time.Time
	// Metadata defines metadata values.
	Metadata map[string]string
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time
}

// OrderItemRow defines flattened order item export values.
type OrderItemRow struct {
	// SKU defines product SKU values.
	SKU string
	// AlternateName defines product display-name values.
	AlternateName string
	// Quantity defines item quantity values.
	Quantity int
	// Value defines item value values.
	Value float64
	// ProductID defines optional product identifiers.
	ProductID string
}

// OrderRow defines flattened order export values.
type OrderRow struct {
	// ID defines order identifier values.
	ID string
	// Identifier defines external order identifiers.
	Identifier string
	// Realm defines order realm values.
	Realm string
	// ContactID defines customer contact identifiers.
	ContactID string
	// ContactEmail defines customer email values.
	ContactEmail string
	// Address defines shipping address line 1 values.
	Address string
	// Address2 defines shipping address line 2 values.
	Address2 string
	// Phone defines shipping phone values.
	Phone string
	// CityName defines human-readable city values when available.
	CityName string
	// CityCode defines city code values.
	CityCode string
	// Status defines current order status values.
	Status string
	// Items defines ordered item values.
	Items []OrderItemRow
	// PaymentMethod defines order payment method values.
	PaymentMethod string
	// Metadata defines metadata values.
	Metadata map[string]string
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time
}

// ContactConsentStatus defines flattened contact consent state values.
type ContactConsentStatus struct {
	// Channel defines consent channel values.
	Channel string
	// Action defines latest consent action values.
	Action string
	// OccurredAt defines latest consent decision timestamps.
	OccurredAt time.Time
}

// ContactSource defines contact export source behavior.
type ContactSource interface {
	// ListContacts returns all contacts to export.
	ListContacts(ctx context.Context) ([]ContactRow, error)
}

// ContactConsentSource defines optional contact consent lookup behavior.
type ContactConsentSource interface {
	// GetContactStatuses returns latest consent statuses for a contact.
	GetContactStatuses(ctx context.Context, contactID string) ([]ContactConsentStatus, error)
}

// OrderSource defines order export source behavior.
type OrderSource interface {
	// ListOrders returns all orders to export.
	ListOrders(ctx context.Context) ([]OrderRow, error)
}
