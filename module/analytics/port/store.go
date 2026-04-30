package port

import (
	"context"
	"time"

	"mannaiah/module/analytics/domain"
)

// ContactSnapshot defines analytical contact snapshot row values.
type ContactSnapshot struct {
	// ContactID defines contact identifier values.
	ContactID string
	// Email defines contact email values.
	Email string
	// FirstName defines contact first-name values.
	FirstName string
	// LastName defines contact last-name values.
	LastName string
	// LegalName defines contact legal-name values.
	LegalName string
	// Phone defines contact phone values.
	Phone string
	// CityCode defines contact city-code values.
	CityCode string
	// DocumentType defines contact document-type values.
	DocumentType string
	// Metadata defines optional metadata values.
	Metadata map[string]string
	// CreatedAt defines creation timestamp values.
	CreatedAt time.Time
	// UpdatedAt defines update timestamp values.
	UpdatedAt time.Time
}

// OrderFact defines analytical order fact row values.
type OrderFact struct {
	// OrderID defines order identifier values.
	OrderID string
	// Identifier defines external order identifier values.
	Identifier string
	// Realm defines order realm values.
	Realm string
	// ContactID defines contact identifier values.
	ContactID string
	// CurrentStatus defines current order-status values.
	CurrentStatus string
	// TotalValue defines order total-value values.
	TotalValue float64
	// ItemCount defines total order-item count values.
	ItemCount int
	// CreatedAt defines order creation timestamp values.
	CreatedAt time.Time
	// UpdatedAt defines order update timestamp values.
	UpdatedAt time.Time
}

// OrderItemFact defines analytical order-item fact row values.
type OrderItemFact struct {
	// OrderID defines order identifier values.
	OrderID string
	// ContactID defines contact identifier values.
	ContactID string
	// SKU defines SKU values.
	SKU string
	// AlternateName defines alternate-name values.
	AlternateName string
	// ProductID defines optional product identifier values.
	ProductID string
	// Quantity defines ordered quantity values.
	Quantity int
	// Value defines item value values.
	Value float64
	// ResolutionSource defines item resolution-source values.
	ResolutionSource string
	// OrderCreatedAt defines order creation timestamp values.
	OrderCreatedAt time.Time
	// OrderUpdatedAt defines order update timestamp values.
	OrderUpdatedAt time.Time
}

// MembershipEvent defines analytical membership-event row values.
type MembershipEvent struct {
	// ContactID defines contact identifier values.
	ContactID string
	// Channel defines channel values.
	Channel string
	// Action defines action values.
	Action string
	// Source defines source values.
	Source string
	// OccurredAt defines event timestamp values.
	OccurredAt time.Time
}

// Store defines analytics storage behavior.
type Store interface {
	// Ping verifies analytics backend connectivity.
	Ping(ctx context.Context) error
	// EnsureSchema applies analytical schema dependencies.
	EnsureSchema(ctx context.Context) error
	// UpsertContacts upserts contact snapshot rows.
	UpsertContacts(ctx context.Context, rows []ContactSnapshot) error
	// UpsertOrders upserts order fact rows.
	UpsertOrders(ctx context.Context, rows []OrderFact) error
	// UpsertOrderItems appends order item fact rows.
	UpsertOrderItems(ctx context.Context, rows []OrderItemFact) error
	// InsertMembershipEvents inserts membership event rows.
	InsertMembershipEvents(ctx context.Context, rows []MembershipEvent) error
	// ResolveContacts resolves analytical contact IDs by filter.
	ResolveContacts(ctx context.Context, filter domain.SegmentFilter, page int, limit int) ([]string, error)
	// CountContacts counts analytical contacts by filter.
	CountContacts(ctx context.Context, filter domain.SegmentFilter) (int64, error)
}
