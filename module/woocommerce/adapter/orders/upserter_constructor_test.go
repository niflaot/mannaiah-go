package orders

import (
	errorspkg "errors"
	"testing"
)

// TestNewUpserterValidation verifies constructor validation behavior.
func TestNewUpserterValidation(t *testing.T) {
	contactService := newContactServiceMock()
	orderService := newOrdersServiceMock()

	if _, err := NewUpserter(nil, contactService); !errorspkg.Is(err, ErrNilOrderService) {
		t.Fatalf("NewUpserter(nil orderService) error = %v, want ErrNilOrderService", err)
	}
	if _, err := NewUpserter(orderService, nil); !errorspkg.Is(err, ErrNilContactService) {
		t.Fatalf("NewUpserter(nil contactService) error = %v, want ErrNilContactService", err)
	}
}
