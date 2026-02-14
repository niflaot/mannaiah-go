package port

import (
	"context"
	"errors"

	ordersdomain "mannaiah/module/orders/domain"
)

var (
	// ErrNotFound is returned when order rows are not found.
	ErrNotFound = errors.New("order not found")
	// ErrDuplicateIdentifier is returned when realm+identifier already exists.
	ErrDuplicateIdentifier = errors.New("order identifier already exists for realm")
)

// ListQuery defines order-list filtering values.
type ListQuery struct {
	// Page defines requested page values.
	Page int
	// Limit defines requested page-size values.
	Limit int
	// Realm defines optional realm filter values.
	Realm string
	// ContactID defines optional contact-id filter values.
	ContactID string
	// Identifier defines optional identifier filter values.
	Identifier string
	// Status defines optional status filter values.
	Status ordersdomain.Status
}

// Repository defines order persistence behavior.
type Repository interface {
	// Create persists order aggregate values.
	Create(ctx context.Context, order *ordersdomain.Order) error
	// GetByID retrieves order aggregate values by identifier.
	GetByID(ctx context.Context, id string) (*ordersdomain.Order, error)
	// List retrieves paginated order rows with total values.
	List(ctx context.Context, query ListQuery) (rows []ordersdomain.Order, total int64, err error)
	// AppendStatus appends status rows and updates current status values.
	AppendStatus(ctx context.Context, id string, entry ordersdomain.StatusEntry) (*ordersdomain.Order, error)
}
