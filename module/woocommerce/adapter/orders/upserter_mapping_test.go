package orders

import (
	"testing"
	"time"

	ordersdomain "mannaiah/module/orders/domain"
	"mannaiah/module/woocommerce/port"
)

// TestToCreateCommandMapsPaymentMethod verifies payment method propagation into order creation commands.
func TestToCreateCommandMapsPaymentMethod(t *testing.T) {
	createdAt := time.Date(2026, time.March, 20, 12, 0, 0, 0, time.UTC)
	command := toCreateCommand(
		port.OrderSyncCommand{
			Identifier:    "2001",
			Realm:         "woocommerce",
			Status:        "processing",
			PaymentMethod: "  payonline  ",
			CreatedAt:     &createdAt,
			Items:         []port.OrderSyncItem{{SKU: "SKU-1", Quantity: 1}},
		},
		"contact-1",
		"woocommerce",
		ordersdomain.StatusCreated,
		nil,
	)

	if command.PaymentMethod != "payonline" {
		t.Fatalf("command.PaymentMethod = %q, want %q", command.PaymentMethod, "payonline")
	}
	if command.Identifier != "2001" {
		t.Fatalf("command.Identifier = %q, want %q", command.Identifier, "2001")
	}
	if command.Realm != "woocommerce" {
		t.Fatalf("command.Realm = %q, want %q", command.Realm, "woocommerce")
	}
	if command.ContactID != "contact-1" {
		t.Fatalf("command.ContactID = %q, want %q", command.ContactID, "contact-1")
	}
}
