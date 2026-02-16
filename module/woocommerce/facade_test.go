package woocommerce

import (
	"context"
	errorspkg "errors"
	"testing"

	contactapplication "mannaiah/module/contacts/application"
	contactdomain "mannaiah/module/contacts/domain"
	contactport "mannaiah/module/contacts/port"
	ordersapplication "mannaiah/module/orders/application"
	ordersdomain "mannaiah/module/orders/domain"
)

// TestOpenAPISpecFacade verifies root facade OpenAPI delegation behavior.
func TestOpenAPISpecFacade(t *testing.T) {
	spec := OpenAPISpec()
	if spec == nil {
		t.Fatalf("OpenAPISpec() should not return nil")
	}
	if spec.OpenAPI != "3.0.3" {
		t.Fatalf("spec.OpenAPI = %q, want %q", spec.OpenAPI, "3.0.3")
	}
}

// TestNewFacadeValidation verifies root facade constructor validation behavior.
func TestNewFacadeValidation(t *testing.T) {
	if _, err := New(Config{}, nil, orderServiceMock{}, nil, nil); !errorspkg.Is(err, ErrNilContactService) {
		t.Fatalf("New() error = %v, want ErrNilContactService", err)
	}
	if _, err := New(Config{}, contactServiceMock{}, nil, nil, nil); !errorspkg.Is(err, ErrNilOrderService) {
		t.Fatalf("New() error = %v, want ErrNilOrderService", err)
	}
}

// contactServiceMock defines contacts service behavior for facade tests.
type contactServiceMock struct{}

// Create creates contacts.
func (contactServiceMock) Create(ctx context.Context, command contactapplication.CreateCommand) (*contactdomain.Contact, error) {
	return &contactdomain.Contact{ID: "contact-1", Email: command.Email}, nil
}

// Get retrieves contacts.
func (contactServiceMock) Get(ctx context.Context, id string) (*contactdomain.Contact, error) {
	return &contactdomain.Contact{ID: id, Email: "test@example.com"}, nil
}

// List lists contacts.
func (contactServiceMock) List(ctx context.Context, query contactport.ListQuery) (*contactapplication.ListResult, error) {
	return &contactapplication.ListResult{}, nil
}

// Update updates contacts.
func (contactServiceMock) Update(ctx context.Context, id string, command contactapplication.UpdateCommand) (*contactdomain.Contact, error) {
	return &contactdomain.Contact{ID: id, Email: "test@example.com"}, nil
}

// Delete deletes contacts.
func (contactServiceMock) Delete(ctx context.Context, id string) error { return nil }

// orderServiceMock defines orders service behavior for facade tests.
type orderServiceMock struct{}

// Create creates orders.
func (orderServiceMock) Create(ctx context.Context, command ordersapplication.CreateCommand) (*ordersdomain.Order, error) {
	return &ordersdomain.Order{}, nil
}

// Get retrieves orders.
func (orderServiceMock) Get(ctx context.Context, id string) (*ordersdomain.Order, error) {
	return &ordersdomain.Order{}, nil
}

// List lists orders.
func (orderServiceMock) List(ctx context.Context, query ordersapplication.ListQuery) (*ordersapplication.ListResult, error) {
	return &ordersapplication.ListResult{}, nil
}

// Update updates mutable order rows.
func (orderServiceMock) Update(ctx context.Context, id string, command ordersapplication.UpdateCommand) (*ordersdomain.Order, error) {
	return &ordersdomain.Order{}, nil
}

// UpdateStatus updates order status rows.
func (orderServiceMock) UpdateStatus(ctx context.Context, id string, command ordersapplication.UpdateStatusCommand) (*ordersdomain.Order, error) {
	return &ordersdomain.Order{}, nil
}

// AddComment appends order comment rows.
func (orderServiceMock) AddComment(ctx context.Context, id string, command ordersapplication.AddCommentCommand) (*ordersdomain.Order, error) {
	return &ordersdomain.Order{}, nil
}

// UpdateComment updates order comment rows.
func (orderServiceMock) UpdateComment(ctx context.Context, id string, commentID string, command ordersapplication.UpdateCommentCommand) (*ordersdomain.Order, error) {
	return &ordersdomain.Order{}, nil
}

// DeleteComment deletes order comment rows.
func (orderServiceMock) DeleteComment(ctx context.Context, id string, commentID string, command ordersapplication.DeleteCommentCommand) (*ordersdomain.Order, error) {
	return &ordersdomain.Order{}, nil
}
