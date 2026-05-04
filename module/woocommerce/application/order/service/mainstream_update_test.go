package service

import (
	"context"
	"errors"
	"testing"

	ordersport "mannaiah/module/orders/port"
	"mannaiah/module/woocommerce/port"
)

// destinationMock defines WooCommerce destination behavior for mainstream update tests.
type destinationMock struct {
	// validateErr defines validation errors.
	validateErr error
	// updateErr defines update errors.
	updateErr error
	// command defines captured update command values.
	command port.MainstreamOrderUpdateCommand
}

// Validate validates destination availability.
func (m *destinationMock) Validate(ctx context.Context) error {
	return m.validateErr
}

// UpdateOrderFromMainstream captures mainstream update command values.
func (m *destinationMock) UpdateOrderFromMainstream(ctx context.Context, command port.MainstreamOrderUpdateCommand) error {
	m.command = command
	return m.updateErr
}

// TestNewMainstreamUpdateServiceValidation verifies constructor validation behavior.
func TestNewMainstreamUpdateServiceValidation(t *testing.T) {
	if _, err := NewMainstreamUpdateService(nil, nil); !errors.Is(err, ErrNilDestination) {
		t.Fatalf("NewMainstreamUpdateService(nil) error = %v, want ErrNilDestination", err)
	}
}

// TestHandleOrderEventSkipsNonWooAndLoopSources verifies skip behavior.
func TestHandleOrderEventSkipsNonWooAndLoopSources(t *testing.T) {
	destination := &destinationMock{}
	service, err := NewMainstreamUpdateService(destination, nil)
	if err != nil {
		t.Fatalf("NewMainstreamUpdateService() error = %v", err)
	}

	if err := service.HandleOrderEvent(context.Background(), ordersport.OrderEventPayload{Realm: "website", Identifier: "1001"}); err != nil {
		t.Fatalf("HandleOrderEvent(non-woo) error = %v", err)
	}
	if err := service.HandleOrderEvent(context.Background(), ordersport.OrderEventPayload{Realm: "woocommerce", Source: "woocommerce_sync", Identifier: "1001"}); err != nil {
		t.Fatalf("HandleOrderEvent(loop-source) error = %v", err)
	}
	if err := service.HandleOrderEvent(context.Background(), ordersport.OrderEventPayload{Realm: "woocommerce", Identifier: "internal-1"}); err != nil {
		t.Fatalf("HandleOrderEvent(non-numeric-identifier) error = %v", err)
	}
	if destination.command.Identifier != "" {
		t.Fatalf("unexpected command capture: %+v", destination.command)
	}
}

// TestHandleOrderEventMapsCommand verifies payload-to-command mapping behavior.
func TestHandleOrderEventMapsCommand(t *testing.T) {
	destination := &destinationMock{}
	service, err := NewMainstreamUpdateService(destination, nil)
	if err != nil {
		t.Fatalf("NewMainstreamUpdateService() error = %v", err)
	}

	err = service.HandleOrderEvent(context.Background(), ordersport.OrderEventPayload{
		Identifier:    "1001",
		Realm:         "woocommerce",
		CurrentStatus: "CREATED",
		LatestStatus: ordersport.OrderEventStatus{
			Status: "COMPLETED",
		},
		Items: []ordersport.OrderEventItem{
			{SKU: "SKU-1", AlternateName: "Item Name", Quantity: 2, Value: 20},
		},
		ShippingAddress: ordersport.OrderEventShippingAddress{
			Address:  "A",
			Address2: "B",
			Phone:    "300",
			CityCode: "11001",
		},
		ShippingCharges: []ordersport.OrderEventShippingCharge{
			{MethodID: "flat_rate", MethodTitle: "Flat", Price: 10},
		},
	})
	if err != nil {
		t.Fatalf("HandleOrderEvent() error = %v", err)
	}
	if destination.command.Identifier != "1001" {
		t.Fatalf("command.Identifier = %q, want %q", destination.command.Identifier, "1001")
	}
	if destination.command.Status != "completed" {
		t.Fatalf("command.Status = %q, want %q", destination.command.Status, "completed")
	}
	if len(destination.command.Items) != 1 || destination.command.Items[0].SKU != "SKU-1" {
		t.Fatalf("command.Items = %+v, want one SKU-1 row", destination.command.Items)
	}
}

// TestHandleOrderEventErrors verifies error mapping behavior.
func TestHandleOrderEventErrors(t *testing.T) {
	destination := &destinationMock{validateErr: errors.New("validate failed")}
	service, err := NewMainstreamUpdateService(destination, nil)
	if err != nil {
		t.Fatalf("NewMainstreamUpdateService() error = %v", err)
	}

	if err := service.HandleOrderEvent(context.Background(), ordersport.OrderEventPayload{Realm: "woocommerce", Identifier: "1001"}); err == nil {
		t.Fatalf("expected validation error")
	}

	destination = &destinationMock{updateErr: errors.New("update failed")}
	service, err = NewMainstreamUpdateService(destination, nil)
	if err != nil {
		t.Fatalf("NewMainstreamUpdateService() error = %v", err)
	}
	if err := service.HandleOrderEvent(context.Background(), ordersport.OrderEventPayload{Realm: "woocommerce", Identifier: "1001"}); err == nil {
		t.Fatalf("expected update error")
	}
}

// TestIsWooNumericIdentifier verifies WooCommerce numeric identifier checks.
func TestIsWooNumericIdentifier(t *testing.T) {
	if !isWooNumericIdentifier("1001") {
		t.Fatalf("isWooNumericIdentifier(1001) = false, want true")
	}
	if isWooNumericIdentifier("internal-1") {
		t.Fatalf("isWooNumericIdentifier(internal-1) = true, want false")
	}
	if isWooNumericIdentifier("0") {
		t.Fatalf("isWooNumericIdentifier(0) = true, want false")
	}
}

// TestMapMainstreamStatus verifies order status mapping behavior for mainstream updates.
func TestMapMainstreamStatus(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{input: "CREATED", want: "processing"},
		{input: "PENDING", want: "pending"},
		{input: "HOLD", want: "on-hold"},
		{input: "COMPLETED", want: "completed"},
		{input: "CANCELLED", want: "cancelled"},
		{input: "UNKNOWN", want: ""},
	}

	for _, tc := range cases {
		if got := mapMainstreamStatus(tc.input); got != tc.want {
			t.Fatalf("mapMainstreamStatus(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
