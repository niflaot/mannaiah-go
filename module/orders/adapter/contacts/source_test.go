package contacts

import (
	"context"
	"errors"
	"testing"

	contactsapplication "mannaiah/module/contacts/application"
	contactsdomain "mannaiah/module/contacts/domain"
	contactsport "mannaiah/module/contacts/port"
	ordersport "mannaiah/module/orders/port"
)

// serviceMock defines contacts service behavior for source tests.
type serviceMock struct {
	// getFn defines get behavior.
	getFn func(ctx context.Context, id string) (*contactsdomain.Contact, error)
}

// Create executes mocked create behavior.
func (m serviceMock) Create(ctx context.Context, command contactsapplication.CreateCommand) (*contactsdomain.Contact, error) {
	return nil, nil
}

// Get executes mocked get behavior.
func (m serviceMock) Get(ctx context.Context, id string) (*contactsdomain.Contact, error) {
	return m.getFn(ctx, id)
}

// List executes mocked list behavior.
func (m serviceMock) List(ctx context.Context, query contactsport.ListQuery) (*contactsapplication.ListResult, error) {
	return nil, nil
}

// Update executes mocked update behavior.
func (m serviceMock) Update(ctx context.Context, id string, command contactsapplication.UpdateCommand) (*contactsdomain.Contact, error) {
	return nil, nil
}

// Delete executes mocked delete behavior.
func (m serviceMock) Delete(ctx context.Context, id string) error {
	return nil
}

// TestNewSourceValidation verifies constructor validation behavior.
func TestNewSourceValidation(t *testing.T) {
	_, err := NewSource(nil)
	if !errors.Is(err, ErrNilService) {
		t.Fatalf("NewSource(nil) error = %v, want ErrNilService", err)
	}
}

// TestGetByID verifies customer lookup mapping behavior.
func TestGetByID(t *testing.T) {
	source, err := NewSource(serviceMock{
		getFn: func(ctx context.Context, id string) (*contactsdomain.Contact, error) {
			return &contactsdomain.Contact{
				ID:           id,
				Address:      "Street 1",
				AddressExtra: "Apt 1",
				Phone:        "+573001112233",
				CityCode:     "110111",
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("NewSource() error = %v", err)
	}

	customer, err := source.GetByID(context.Background(), "c-1")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if customer.ID != "c-1" {
		t.Fatalf("customer.ID = %q, want %q", customer.ID, "c-1")
	}
}

// TestGetByIDErrorMapping verifies error mapping behavior.
func TestGetByIDErrorMapping(t *testing.T) {
	source, err := NewSource(serviceMock{
		getFn: func(ctx context.Context, id string) (*contactsdomain.Contact, error) {
			return nil, contactsport.ErrNotFound
		},
	})
	if err != nil {
		t.Fatalf("NewSource() error = %v", err)
	}

	_, err = source.GetByID(context.Background(), "c-1")
	if !errors.Is(err, ordersport.ErrCustomerNotFound) {
		t.Fatalf("GetByID() error = %v, want ErrCustomerNotFound", err)
	}
}
