package domain

import "time"

const (
	// EventTypeContactCreated defines contact-created domain event types.
	EventTypeContactCreated = "contacts.contact.created"
	// EventTypeContactUpdated defines contact-updated domain event types.
	EventTypeContactUpdated = "contacts.contact.updated"
)

// Event defines shared behavior for contact domain events.
type Event interface {
	// Name returns the event type name.
	Name() string
	// AggregateID returns the contact aggregate identifier.
	AggregateID() string
	// OccurredAt returns when the event occurred.
	OccurredAt() time.Time
}

// ContactCreatedEvent defines domain events emitted after contact creation.
type ContactCreatedEvent struct {
	// Contact defines the aggregate snapshot.
	Contact Contact
	// At defines when the event occurred.
	At time.Time
}

// NewContactCreatedEvent creates contact-created domain events.
func NewContactCreatedEvent(contact Contact) ContactCreatedEvent {
	occurredAt := contact.CreatedAt
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	return ContactCreatedEvent{
		Contact: contact,
		At:      occurredAt,
	}
}

// Name returns the event type name.
func (e ContactCreatedEvent) Name() string {
	return EventTypeContactCreated
}

// AggregateID returns the contact aggregate identifier.
func (e ContactCreatedEvent) AggregateID() string {
	return e.Contact.ID
}

// OccurredAt returns when the event occurred.
func (e ContactCreatedEvent) OccurredAt() time.Time {
	return e.At
}

// ContactUpdatedEvent defines domain events emitted after contact updates.
type ContactUpdatedEvent struct {
	// Contact defines the aggregate snapshot.
	Contact Contact
	// At defines when the event occurred.
	At time.Time
}

// NewContactUpdatedEvent creates contact-updated domain events.
func NewContactUpdatedEvent(contact Contact) ContactUpdatedEvent {
	occurredAt := contact.UpdatedAt
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	return ContactUpdatedEvent{
		Contact: contact,
		At:      occurredAt,
	}
}

// Name returns the event type name.
func (e ContactUpdatedEvent) Name() string {
	return EventTypeContactUpdated
}

// AggregateID returns the contact aggregate identifier.
func (e ContactUpdatedEvent) AggregateID() string {
	return e.Contact.ID
}

// OccurredAt returns when the event occurred.
func (e ContactUpdatedEvent) OccurredAt() time.Time {
	return e.At
}
