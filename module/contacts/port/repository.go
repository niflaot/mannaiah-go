package port

import (
	"context"
	"errors"

	"mannaiah/module/contacts/domain"
)

var (
	// ErrNotFound is returned when a contact is not found.
	ErrNotFound = errors.New("contact not found")
	// ErrDuplicateContact is returned when a contact violates uniqueness constraints.
	ErrDuplicateContact = errors.New("contact already exists")
	// ErrDuplicateEmail is returned when a contact already uses the provided email.
	ErrDuplicateEmail = errors.New("contact email already exists")
	// ErrDuplicateDocument is returned when a contact already uses the same document type and number.
	ErrDuplicateDocument = errors.New("contact document already exists")
)

// ListQuery defines query-side filters for contact listing.
type ListQuery struct {
	// Page defines requested page number.
	Page int
	// Limit defines requested page size.
	Limit int
	// OrderBy defines sortable fields.
	OrderBy string
	// OrderDir defines sort direction: asc or desc.
	OrderDir string
	// Email defines optional email filter.
	Email string
	// ExcludeIDs defines ids excluded from result and total count.
	ExcludeIDs []string
	// MetadataKey defines optional metadata-key filters.
	MetadataKey string
	// MetadataValue defines optional metadata-value filters.
	MetadataValue string
}

// Repository defines persistence behavior required by contact use cases.
type Repository interface {
	// Create persists a new contact entity.
	Create(ctx context.Context, contact *domain.Contact) error
	// GetByID retrieves a contact entity by id.
	GetByID(ctx context.Context, id string) (*domain.Contact, error)
	// List retrieves contacts and total count using query-side filters.
	List(ctx context.Context, query ListQuery) ([]domain.Contact, int64, error)
	// Update persists modifications for an existing contact.
	Update(ctx context.Context, contact *domain.Contact) error
	// Delete soft-deletes a contact by id.
	Delete(ctx context.Context, id string) error
}
